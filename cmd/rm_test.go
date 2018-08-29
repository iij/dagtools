package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"github.com/golang/mock/gomock"
)

func TestRmUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(rmCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestRmAnBucket(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(rmCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().DeleteBucket("mybucket").Return(errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("mybucket"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestRmAnBucketRecursive(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(rmCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	listing := client.ObjectListing{
		Summaries: []client.ObjectSummary{{Key: "dummy"}},
	}
	mock.EXPECT().ListObjects("mybucket", "", "", "", 1000).Return(&listing, nil)
	m := client.MultipleDeletionResult{
		DeletedObjects: []client.DeletedObject{{Key: "dummy"}},
	}
	mock.EXPECT().DeleteMultipleObjects("mybucket", []string{"dummy"}, false).Return(&m, nil)
	mock.EXPECT().DeleteBucket("mybucket").Return(errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("-r mybucket"))
	if !c.recursive {
		t.Error("rmCommand::recursive does not true.")
	}
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}
