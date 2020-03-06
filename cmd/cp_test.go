package cmd

import (
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"strings"
	"testing"
	"time"
)

func TestCpUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cp command usage. usage: %q", usage)
	}
}

func TestCpObjectToBucket(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().PutObjectCopy("mybucket", "test/myobject", "mybucket2", "myobject", nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs("mybucket:test/myobject mybucket2:"))
	if err != nil {
		t.Errorf("Should return nil. %v", err)
	}
}

func TestCpObjectToDir(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().PutObjectCopy("mybucket", "test/myobject", "mybucket2", "test2/myobject", nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs("mybucket:test/myobject mybucket2:test2/"))
	if err != nil {
		t.Errorf("Should return nil. %v", err)
	}
}

func TestCpDirToBucket(t *testing.T) {
	listing := &client.ObjectListing{Name: "", Location: "ap2", Prefix: "", Marker: "", MaxKeys: 1000, Delimiter: "/", NextMarker: "", IsTruncated: false,
		Summaries:      []client.ObjectSummary{{"test/object", time.Now(), "", int64(100), "", client.Owner{ID: "123", DisplayName: "hoge"}}},
		CommonPrefixes: []client.CommonPrefix{{"test/"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects("mybucket", "test/", "", "/", 1000).Return(listing, nil)
	mock.EXPECT().PutObjectCopy("mybucket", listing.Summaries[0].Key, "mybucket2", listing.Summaries[0].Key, nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs("-r mybucket:test/ mybucket2:"))
	if err != nil {
		t.Errorf("Should return nil. %v", err)
	}
}

func TestCpDirToDir(t *testing.T) {
	listing := &client.ObjectListing{Name: "", Location: "ap2", Prefix: "", Marker: "", MaxKeys: 1000, Delimiter: "/", NextMarker: "", IsTruncated: false,
		Summaries:      []client.ObjectSummary{{"test/object", time.Now(), "", int64(100), "", client.Owner{ID: "123", DisplayName: "hoge"}}},
		CommonPrefixes: []client.CommonPrefix{{"test/"}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects("mybucket", "test/", "", "/", 1000).Return(listing, nil)
	mock.EXPECT().PutObjectCopy("mybucket", listing.Summaries[0].Key, "mybucket2", "test2/test/object", nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs("-r mybucket:test/ mybucket2:test2/"))
	if err != nil {
		t.Errorf("Should return nil. %v", err)
	}
}

func TestCpBucketToBucket(t *testing.T) {
	listing := &client.ObjectListing{Name: "", Location: "ap2", Prefix: "", Marker: "", MaxKeys: 1000, Delimiter: "/", NextMarker: "", IsTruncated: false,
		Summaries:      []client.ObjectSummary{{"test/object", time.Now(), "", int64(100), "", client.Owner{ID: "123", DisplayName: "hoge"}}},
		CommonPrefixes: []client.CommonPrefix{{""}},
	}
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects("mybucket", "", "", "", 1000).Return(listing, nil)
	mock.EXPECT().PutObjectCopy("mybucket", listing.Summaries[0].Key, "mybucket2", listing.Summaries[0].Key, nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs("-r mybucket: mybucket2:"))
	if err != nil {
		t.Errorf("Should return err. %v", err)
	}
}

func TestCpOptionErrBucket(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock

	err := c.Run(parseArgs("mybucket: mybucket2:"))
	if err == nil {
		t.Errorf("Should return ErrArgument. %v", err)
	}
	if err != nil {
		fmt.Println(err.Error())
	}
}

func TestCpOptionErrDir(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(cpCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	c.cli = mock

	err := c.Run(parseArgs("mybucket:test/ mybucket2:test2/"))
	if err == nil {
		t.Errorf("Should return ErrArgument. %v", err)
	}
	if err != nil {
		fmt.Println(err.Error())
	}
}
