package main

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
)

func TestCommitId(t *testing.T) {
	cases := []struct {
		input string
		id    string
	}{
		{"commit d0532bdb9ab40e06ee0702481f623d5054c8831a", "d0532bdb9ab40e06ee0702481f623d5054c8831a"},
		{"commit d0532bdb9ab40e06ee0702481f623d5054c8831a\n", "d0532bdb9ab40e06ee0702481f623d5054c8831a"},
		{"commit d0532bdb9ab40e06ee0702481f623d5054c8831a", "d0532bdb9ab40e06ee0702481f623d5054c8831a"},
	}

	for _, sample := range cases {
		t.Run("", func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(getCommitID(sample.input)).Should(Equal(sample.id))
		})
	}
}

func TestAuthor(t *testing.T) {
	cases := []struct {
		input string
		id    string
	}{
		{"Author: Albert <albert@gmail.com>", "Albert <albert@gmail.com>"},
	}

	for _, sample := range cases {
		t.Run("", func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(getAuthor(sample.input)).Should(Equal(sample.id))
		})
	}
}

func TestForGetLastEmptyLine(t *testing.T){
	cases := []struct {
		input []string
		line int 
	}{
		{[]string{"", "1\t2\tstaging/src/k8s.io/apiserver/pkg/server/filters/BUILD"}, 0},
		{[]string{"1\t2\tstaging/src/k8s.io/apiserver/pkg/server/filters/BUILD",""}, 1},
		{[]string{"", "", "1\t2\tstaging/src/k8s.io/apiserver/pkg/server/filters/BUILD"}, 1},
		{[]string{"1\t2\tstaging/src/k8s.io/apiserver/pkg/server/filters/BUILD"}, -1},
		{[]string{"commit d0532bdb9ab40e06ee0702481f623d5054c8831a",
		"Author: Han Kang <hankang@google.com>",
		"Date:   2019-01-04 14:06:46 -0800",
		"",
		"    add a content-type filter to apiserver filters to autoset nosniff", 
		"",
		"1\t2\tstaging/src/k8s.io/apiserver/pkg/server/filters/BUILD"}, 5},
		
	}
	for _, sample := range cases {
		t.Run("", func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(getLastEmptyLine(sample.input)).To(Equal(sample.line))
		})
	}
	
}

func TestTime(t *testing.T) {
	cases := []struct {
		input string
		t     time.Time
	}{
		{"Date:   2015-12-21 18:15:30 -0100", time.Date(2015, 12, 21, 19, 15, 30, 0, time.UTC)},
		{"Date:   2015-12-21 18:15:30 -0000", time.Date(2015, 12, 21, 18, 15, 30, 0, time.UTC)},
	}

	for _, sample := range cases {
		t.Run("", func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(getTime(sample.input)).To(BeTemporally("==", sample.t))
		})
	}
}

func TestForFirstLineOfCommit(t *testing.T) {
	cases := []struct {
		input string
		firstLine bool     
	}{
		
		{"    This reverts commit 72792d59f46f822cf360e797d886e582a6a2dc60.", false},
		{"commit 72792d59f46f822cf360e797d886e582a6a2dc60", true},
	}

	for _, sample := range cases {
		t.Run("", func(t *testing.T) {
			g := NewGomegaWithT(t)
			g.Expect(isFirstLineOfCommit(sample.input)).To(Equal(sample.firstLine))
		})
	}
}
