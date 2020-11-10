package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
)

func TestCatUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(catCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestCatAnObject(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(catCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetObject("mybucket", "foo/bar").Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("mybucket:foo/bar"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestCatAnObjectWithIllegalArgument(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(catCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock
	err := c.Run(parseArgs("mybucket"))
	if err != ErrArgument {
		t.Errorf("%v != %v", ErrArgument, err)
	}
	err = c.Run(parseArgs("mybucket:aaa mybucket:bbb"))
	if err != ErrArgument {
		t.Errorf("%v != %v", ErrArgument, err)
	}
}

func TestCatAnObjectWithSpecialKey(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(catCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetObject("mybucket", "foo:bar").Return(nil, errors.New("ok"))
	mock.EXPECT().GetObject("mybucket", "日本語/にほんご").Return(nil, errors.New("ok"))
	c.cli = mock
	err := c.Run(parseArgs("mybucket:foo:bar"))
	if err.Error() != "ok" {
		t.Error("Error message was not match. ", err)
	}
	err = c.Run(parseArgs("mybucket:日本語/にほんご"))
	if err.Error() != "ok" {
		t.Error("Error message was not match. ", err)
	}
}
