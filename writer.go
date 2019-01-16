package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type CommitDBWriter struct {
	buf []Commit
	db  sql.DB
}

func NewCommitDBWriter() CommitDBWriter {
	return CommitDBWriter{buf: make([]Commit, 0)}
}

func (w *CommitDBWriter) Consume(c Commit) error {
	fmt.Println(c)
	return nil
}

func (w *CommitDBWriter) Init() error {
	os.Remove("foo.db")
	db, err := sql.Open("sqlite3", "./foo.db")

	if err != nil {
		return errors.New("failed to initiate db")
	}
	w.db = *db

	sqlStmt := `
	create table commits(id string not null primary key, author text, comments text, created integer);
	create table changes(id string not null primary key, added integer, removed integer, string file);	
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		return errors.Wrap(err, "failed to create tables")
	}
	return nil
}

func (w *CommitDBWriter) Close() error {
	return w.db.Close()
}
