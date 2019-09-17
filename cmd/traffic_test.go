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

func TestTrafficUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestGetTrafficOfDay(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetNetworkTraffic("20151020", gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("20151020"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.humanReadable {
		t.Error(`Did not specify "-h" option`)
	}
}

func TestListTraffics(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListNetworkTraffics(2, gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("-b 2"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.humanReadable {
		t.Error(`Did not specify "-h" option`)
	}
}

func TestCallTrafficCommandWithNoArguments(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock
	err := c.Run(parseArgs(""))
	if err != ErrArgument {
		t.Errorf("Failed to catch an error(ErrArgument)")
	}
}

func TestCallTrafficCommandWithOptions(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetNetworkTraffic("20151020", gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("-h -b 2 20151020"))
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

func TestTrafficWithRegion(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetNetworkTraffic("20151020", gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock

	err := c.Run(parseArgs("-region=ap1 20151020"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
	if c.region != "ap1" {
		t.Errorf("Failed to get region value. ap1 != %v", c.region)
	}
}

func TestTrafficWithTotal(t *testing.T) {
	regions := &client.Regions{
		Regions: []client.Region{{"ap1", "", "ap1.dag.iijgio.com"},
			{"ap2", "", "ap2.dag.iijgio.com"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().GetRegions().Return(regions, nil)
	mock.EXPECT().GetNetworkTraffic("20151020", gomock.Any()).Return(nil, errors.New("dummy"))
	c.cli = mock

	err := c.Run(parseArgs("-t 20151020"))
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

func TestTrafficOptionErr(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(trafficCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock

	err := c.Run(parseArgs("-t -region=ap1 20151020"))
	if err == nil {
		t.Errorf("Failed to return option error")
	}
}
