package main

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type Commit struct {
	headers Headers
	changes []Change
	comment string
}

type Headers struct {
	id     string
	author string
	date   time.Time
}

type Change struct {
	added   uint32
	removed uint32
	path    string
}

//LogReader is a support struct for traversing thru git log
type LogReader struct {
	buf      []string
	line     uint32
	consumer commitConsumer
}

type commitConsumer = func(c Commit) error

func NewLogReader(c commitConsumer) *LogReader {
	return &LogReader{buf: make([]string, 0), line: 0, consumer: c}
}

func (r *LogReader) error(e error) error {
	line := r.line - uint32(len(r.buf))
	return errors.Wrapf(e, "Error processing commit at line %v", line)
}

func (r *LogReader) ReadLine(s string) (err error) {
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
		r.buf = make([]string, 0)
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

func (r LogReader) processCommit() (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			if e, ok := rec.(error); ok {
				err = errors.Wrap(e, "panic")
			} else {
				err = errors.Errorf("%v", rec)
			}
		}
	}()

	r.buf = removeLastEmptyLine(r.buf)	
	c, err := getCommit(r.buf)
	if err != nil {
		return err
	}
	err = r.consumer(*c)
	if err != nil {
		errors.Wrap(err, "failed to consume commit")
	}
	return nil
}

func removeLastEmptyLine(lines []string) []string {
	length := len(lines)
	if len(lines[length-1]) == 0 {
		return lines[:length-1]
	}
	return lines
}

func getCommit(c []string) (*Commit, error) {
	h, err := headersFromLog(c[0:3])
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract headers")
	}

	length := len(c)
	if length == 4 {
		//nor stats nor message are present
		return &Commit{headers: *h}, nil
	}

	statsArePresent, err := regexp.MatchString(`^[\d-]+\t[\d-]+\t`, c[length-1])
	if err != nil {
		return nil, errors.Wrap(err, "failed to lookup for stats")
	}
	var changes []Change
	if statsArePresent {
		lastEmptyLine := getLastEmptyLine(c)
		if lastEmptyLine == -1 {
			return nil, errors.New("invalid commit format")
		}
		changes, err = getChanges(c[lastEmptyLine+1:])
		if err != nil {
			return nil, errors.Wrap(err, "couldn't extract change data")
		}
	}

	var comment string
	linesWithoutStats := length - len(changes)
	commentsArePresent := linesWithoutStats > 4

	if commentsArePresent {
		comment = getComment(c[4:linesWithoutStats])
	}

	return &Commit{headers: *h, comment: comment, changes: changes}, nil
}

func getComment(c []string) string {
	comments := make([]string, 0)
	for _, line := range c {
		if len(line) > 0 {
			comments = append(comments, strings.TrimLeft(line, " "))
		}
	}
	return strings.Join(comments, "\r\n")
}

func getChanges(c []string) ([]Change, error) {
	changes := make([]Change, 0)
	for _, line := range c {
		if line[0:1] == "-" {
			//stats for binary files are omitted
			continue
		}

		c, err := getChange(line)
		if err != nil {
			return changes, errors.Wrap(err, "couldn't extract change data")
		}
		changes = append(changes, *c)
	}
	return changes, nil
}

func getChange(line string) (*Change, error) {
	r, _ := regexp.Compile(`(\d+)\t(\d+)\t(.*)`)

	matches := r.FindStringSubmatch(line)
	if matches == nil {
		return nil, errors.Errorf("invalid stats string: %v", line)
	}

	added, err := getNumericStat(matches[1])
	if err != nil {
		return nil, err
	}

	removed, err := getNumericStat(matches[2])
	if err != nil {
		return nil, err
	}

	return &Change{
		added:   added,
		removed: removed,
		path:    matches[3],
	}, nil

}

func getNumericStat(s string) (uint32, error) {
	i, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to read numeric: %v", s)
	}
	return uint32(i), nil
}

func getLastEmptyLine(commit []string) int {
	for i := len(commit) - 1; i >= 0; i-- {
		line := commit[i]
		if len(line) == 0 {
			return i
		}
	}
	return -1
}

func headersFromLog(text []string) (*Headers, error) {
	timeString := text[2]
	date, err := getTime(timeString)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to extract time from: %v", timeString)
	}

	return &Headers{id: getCommitID(text[0]),
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