package main

import (
	"database/sql"
	"fmt"
	"os"
	"sync"
	"time"

	gosqlite "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type commitDTO struct {
	id      string
	author  string
	date    time.Time
	comment string
}

type changeDTO struct {
	commitID string
	added    uint32
	removed  uint32
	path     string
}

type CommitDBWriter struct {
	db             sql.DB
	commitsChannel chan commitDTO
	changesChannel chan changeDTO
	ready          bool
	waitGroup      sync.WaitGroup
	commits        uint64
	changes        uint64
	dbConnections  []*gosqlite.SQLiteConn
}

func NewCommitDBWriter() CommitDBWriter {
	return CommitDBWriter{
		commitsChannel: make(chan commitDTO, 100),
		changesChannel: make(chan changeDTO, 100),
	}
}

func (w *CommitDBWriter) Consume(c Commit) error {
	if !w.ready {
		return errors.New("Not ready")
	}

	cd := commitDTO{
		author:  c.headers.author,
		comment: c.comment,
		date:    c.headers.date,
		id:      c.headers.id,
	}

	w.commits++
	w.commitsChannel <- cd

	if c.changes == nil {
		return nil
	}
	for _, ch := range c.changes {
		chd := changeDTO{
			added:    ch.added,
			removed:  ch.removed,
			path:     ch.path,
			commitID: c.headers.id,
		}
		w.changes++
		w.changesChannel <- chd
	}
	return nil
}

func (w *CommitDBWriter) Init() error {
	os.Remove("foo.db")

	sql.Register("sqlite3-mem", &gosqlite.SQLiteDriver{
		ConnectHook: func(conn *gosqlite.SQLiteConn) error {
			w.dbConnections = append(w.dbConnections, conn)
			return nil
		},
	})

	db, err := sql.Open("sqlite3-mem", "file::memory:?mode=memory&cache=shared")

	if err != nil {
		return errors.New("failed to initiate db")
	}
	w.db = *db

	sqlStmt := `
	create table commits(id string not null primary key, author text, comments text, created integer);
	create table changes(commit_id string, added integer, removed integer, file string);	
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return errors.Wrap(err, "failed to create tables")
	}

	w.ready = true
	w.waitGroup.Add(4)
	go func() {
		defer w.waitGroup.Done()
		stmt, err := db.Prepare("insert into commits(id, author, comments, created) values(?, ?, ?, ?)")
		if err != nil {
			fmt.Printf("%+v\n", errors.Wrap(err, "failed to prepare connection"))
			w.ready = false
			return
		}
		defer stmt.Close()

		for c := range w.commitsChannel {
			_, err = stmt.Exec(c.id, c.author, c.comment, c.date.Unix())
			if err != nil {
				fmt.Printf("%+v\n", errors.Wrap(err, "failed to prepare connection"))
				w.ready = false
				return
			}
			fmt.Print("0")
		}
	}()

	changeRoutines := 1
	w.waitGroup.Add(changeRoutines)
	for i := 0; i < changeRoutines; i++ {
		go func() {
			defer w.waitGroup.Done()
			stmt, err := db.Prepare("insert into changes(commit_id, added, removed, file) values(?, ?, ?, ?)")
			if err != nil {
				fmt.Printf("%+v\n", errors.Wrap(err, "failed to prepare connection"))
				w.ready = false
				return
			}
			defer stmt.Close()

			for c := range w.changesChannel {
				_, err = stmt.Exec(c.commitID, c.added, c.removed, c.path)
				if err != nil {
					fmt.Printf("%+v\n", errors.Wrap(err, "failed to prepare connection"))
					w.ready = false
					return
				}
				fmt.Print("-")
			}
		}()
	}
	return nil
}

func (w *CommitDBWriter) Close() error {

	drv := gosqlite.SQLiteDriver{}
	c, err := drv.Open("out.db")
	if err != nil {
		return errors.Wrap(err, "failed to open file based DB")
	}

	backup, err := c.(*gosqlite.SQLiteConn).Backup("main", w.dbConnections[0], "main")
	if err != nil {
		return errors.Wrap(err, "failed to dump in-memory DB")
	}

	done, err := backup.Step(-1)
	if err != nil {
		return errors.Wrap(err, "")
	}
	if !done {
		return errors.New("Failed to backup")
	}

	defer c.Close()	
	fmt.Printf("commits: %v, changes: %v", w.commits, w.changes)

	close(w.commitsChannel)
	close(w.changesChannel)
	w.waitGroup.Wait()
	return w.db.Close()
}
