package cmd

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestHumanReadableBytes(t *testing.T) {
	n := HumanReadableBytes(1024)
	if n != "1.0KB" {
		t.Errorf("1.0KB != %v", n)
	}
	n = HumanReadableBytes(1024 * 1024)
	if n != "1.0MB" {
		t.Errorf("1.0MB != %v", n)
	}
	n = HumanReadableBytes(1024 * 1024 * 1024)
	if n != "1.0GB" {
		t.Errorf("1.0GB != %v", n)
	}
	n = HumanReadableBytes(1024 * 1024 * 1024 * 1024)
	if n != "1.0TB" {
		t.Errorf("1.0TB != %v", n)
	}
	n = HumanReadableBytes(1024 * 1.5)
	if n != "1.5KB" {
		t.Errorf("1.5KB != %v", n)
	}
}

func TestLocalString(t *testing.T) {
	t1 := time.Date(2014, 12, 31, 15, 0, 0, 0, time.UTC)
	actual := LocalTimeString(t1)
	expect := "2015-01-01 00:00:00"
	if actual != expect {
		t.Errorf("%v != %v", expect, actual)
	}
}

type fileMatcher struct {
	x string
}

func (m fileMatcher) Matches(x interface{}) bool {
	_x := x.(*os.File)
	return m.x == _x.Name()
}

func (m fileMatcher) String() string {
	return "is file"
}

func parseArgs(args string) []string {
	_args := strings.Split(args, " ")
	if len(_args) == 1 && _args[0] == "" {
		return []string{}
	}
	return _args
}
