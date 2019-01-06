package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

//log should be produced by "git log --no-merges --numstat --date=iso8601"
func main() {

	f, err := os.Open("big.log")

	if err != nil {
		panic(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			panic(err)
		}
	}()

	s := bufio.NewScanner(f)

	reader := newLogReader()
	for s.Scan() {
		err = reader.readLine(s.Text())
		if err != nil {
			panic(err)
		}
	}

	err = reader.close()
	if err != nil {
		panic(err)
	}

	err = s.Err()
	if err != nil {
		panic(err)
	}
}

//LogReader is a support struct for traversing thru git log
type LogReader struct {
	buf  []string
	line uint32
}

func newLogReader() *LogReader {
	return &LogReader{buf: make([]string, 0), line: 0}
}

func (r *LogReader) error(e error) error {
	msg := fmt.Sprintf("Error processing commit at line %v: %v", r.line-uint32(len(r.buf)), e.Error())
	return errors.New(msg)
}

func (r *LogReader) readLine(s string) (err error) {
	r.line = r.line + 1
	separator, err := isFirstLineOfCommit(s)
	if err != nil {
		return r.error(err)
	}

	if separator && len(r.buf) > 0 {
		err = r.processCommit()
		if err != nil {
			return r.error(err)
		}
	}
	r.buf = append(r.buf, s)
	return nil
}

func (r LogReader) close() error {
	err := r.processCommit()
	if err != nil {
		return r.error(err)
	}
	return nil

}

func (r *LogReader) processCommit() (err error) {
	defer func() {
        if r:= recover(); r!= nil {
            err = fmt.Errorf("Failed to parse commit: %v", r)
        }
    }()
	
	
	h, err := headersFromLog(r.buf[0:3])
	if err != nil {
		return err
	}

	fmt.Printf("%v (%v): %v\r", h.date, h.author, h.id)

	r.buf = make([]string, 0)
	return nil
}

type commit struct {
	headers headers
}

type headers struct {
	id     string
	author string
	date   time.Time
}

func headersFromLog(text []string) (h *headers, err error) {
	date, err := getTime(text[2])
	if err != nil {
		return nil, err
	}

	return &headers{id: getCommitID(text[0]),
		author: getAuthor(text[1]),
		date:   date}, nil
}

func getCommitID(s string) string {
	return s[7:47]
}

func getAuthor(s string) string {
	return strings.TrimSpace(s)[8:]
}

func getTime(s string) (time.Time, error) {
	return time.Parse("2006-01-02 15:04:05 -0700", s[8:])
}

func isFirstLineOfCommit(s string) (bool, error) {
	return regexp.MatchString("^commit\\s[0-9a-f]{40}\\s*$", s)
}
