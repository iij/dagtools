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

func TestPolicyUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestPutPolicy(t *testing.T) {
	var bucket = "mybucket"
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().PutBucketPolicy(bucket, gomock.Any()).Return(errors.New("ok"))
	c.cli = mock
	err := c.Run(parseArgs("put mybucket test_files/test-00.txt"))
	if err.Error() != "ok" {
		t.Error("unknown error:", err)
	}
}

func TestCatPolicy(t *testing.T) {
	var bucket = "mybucket"
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetBucketPolicy(bucket).Return(nil, errors.New("ok"))
	c.cli = mock
	err := c.Run(parseArgs("cat mybucket test_files/test-00.txt"))
	if err.Error() != "ok" {
		t.Error("unknown error:", err)
	}
}

func TestRmPolicy(t *testing.T) {
	var bucket = "mybucket"
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().DeleteBucketPolicy(bucket).Return(errors.New("ok"))
	c.cli = mock
	err := c.Run(parseArgs("rm mybucket test_files/test-00.txt"))
	if err.Error() != "ok" {
		t.Error("unknown error:", err)
	}
}

func TestPolicyUnknownCommand(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	err := c.Run(parseArgs("unknown mybucket test_files/test-00.txt"))
	if err.Error() != `policy's sub-command not found: "unknown"` {
		t.Error("unknown error:", err)
	}
}

func TestPolicyIllegalArgument(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(policyCommand)
	c.Init(&e)
	err := c.Run(parseArgs("put"))
	if err != ErrArgument {
		t.Error("unknown error:", err)
	}
}
