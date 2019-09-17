package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"github.com/golang/mock/gomock"
)

func TestLsUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestLsBuckets(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListBuckets().Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(""))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestLsObjects(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects("mybucket", "", "", "/", 1000).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("mybucket"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestLsOneObject(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().DoesObjectExist("mybucket", "foo").Return(true, "ap1", nil)
	mock.EXPECT().ListObjects("mybucket", "foo", "", "", 1).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs("mybucket:foo"))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestLsNotExistDirectory(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mockresp := client.ObjectListing{}

	mock.EXPECT().ListObjects("mybucket", "", "", "/", 1000).Return(&mockresp, nil)
	c.cli = mock
	err := c.Run(parseArgs("mybucket"))
	if err.Error() != "no such file or directory: \"mybucket:\"" {
		t.Errorf("Error message was not match. \"no such file or directory: \"mybucket:\" != %v", err.Error())
	}
}

func TestLsBucketWithRegion(t *testing.T) {
	listing :=&client.BucketListing{
		Owner: client.Owner{"123", "dummy"},
		Buckets: []client.Bucket{{"mybucket",time.Now(), "ap2"},
		{"mybucket2", time.Now(), "ap1"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListBuckets().Return(listing, nil)
	c.cli = mock

	err := c.Run(parseArgs("-region"))
	if err != nil {
		t.Errorf("Could not print Bucket with region option. %v", err)
	}
}

func TestLsObjectWithRegion(t *testing.T) {
	listing := &client.ObjectListing{Name: "", Location: "ap2", Prefix: "", Marker: "", MaxKeys: 1000, Delimiter: "/", NextMarker: "", IsTruncated: false,
		Summaries: []client.ObjectSummary{{"var/dummy.txt", time.Now(), "", int64(100), "", client.Owner{ID: "123",DisplayName: "hoge"}},},
		CommonPrefixes: []client.CommonPrefix{{"var/"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(lsCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects("mybucket", "var/", "", "/", 1000).Return(listing, nil)
	c.cli = mock

	err := c.Run(parseArgs(fmt.Sprintf("-region mybucket:var/")))
	if err != nil {
		t.Errorf("Could not print Object with region option. %v", err)
	}
}