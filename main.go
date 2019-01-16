package main

import (
	"bufio"
	"fmt"
	"os"

	// "github.com/pkg/errors"
)

//log should be produced by "git log --no-merges --numstat --date=iso8601"
func main() {

	writer := CommitDBWriter{}
	err := writer.Init()
	if err != nil {
		fmt.Printf("%+v\n", err)
		os.Exit(1)
	}
	defer writer.Close()


	f, err := os.Open("example.log")
	if err != nil {
		panic(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			panic(err)
		}
	}()

	s := bufio.NewScanner(f)
	reader := NewLogReader(writer.Consume)
	for s.Scan() {
		err = reader.ReadLine(s.Text())
		if err != nil {
			fmt.Printf("%+v\n", err)
			os.Exit(1)
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
