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

func TestSpaceUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a space command usage. usage: %q", usage)
	}
}

func TestCallSpaceCommand(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetStorageSpace(gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(""))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.humanReadable {
		t.Error(`did not specify "-h" option`)
	}
}

func TestCallSpaceCommandWithOptions(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetStorageSpace(gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("-h"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if !c.humanReadable {
		t.Error(`specify "-h" option`)
	}
}

func TestSpaceWithRegion(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetStorageSpace(gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock

	err := c.Run(parseArgs("-region=ap1"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.region != "ap1" {
		t.Errorf("Failed to get region value. ap1 != %s", c.region)
	}
}

func TestSpaceWithTotal(t *testing.T) {
	regions := &client.Regions{
		Regions: []client.Region{{"ap1", "", "ap1.dag.iijgio.com"},
		{"ap2", "", "ap2.dag.iijgio.com"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetRegions().Return(regions, nil)
	mock.EXPECT().GetStorageSpace(gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock

	err := c.Run(parseArgs("-t"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.total != true {
		t.Errorf("Failed to get total value. true != %v", c.total)
	}
}

func TestSpaceOptionErr(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(spaceCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock

	err := c.Run(parseArgs("-t -region=ap1"))
	if err == nil {
		t.Errorf("Failed to return option error")
	}
}