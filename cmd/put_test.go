package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"github.com/golang/mock/gomock"
)

func TestPutUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestPutFile(t *testing.T) {
	var (
		bucket = "mybucket"
		key    = "test.txt"
		path   = "test_files/test-00.txt"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, key, fileMatcher{path}, nil).Return(errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("%s %s:%s", path, bucket, key)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestPutDirectory(t *testing.T) {
	var (
		bucket = "mybucket"
		prefix = "output"
		dir    = "test_files" + string(os.PathSeparator)
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, prefix+"/test-00.txt", fileMatcher{dir + "test-00.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, prefix+"/test-01.txt", fileMatcher{dir + "test-01.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, prefix+"/test-02.txt", fileMatcher{dir + "test-02.txt"}, nil).Return(errors.New("done"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("-r %s %s:%s", dir, bucket, prefix)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "done" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestPutDirectory2(t *testing.T) {
	var (
		bucket = "mybucket"
		prefix = "output/"
		dir    = "test_files" + string(os.PathSeparator)
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-00.txt", fileMatcher{dir + "test-00.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-01.txt", fileMatcher{dir + "test-01.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-02.txt", fileMatcher{dir + "test-02.txt"}, nil).Return(errors.New("done"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("-r %s %s:%s", dir, bucket, prefix)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "done" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestPutDirectoryWithDotSlash(t *testing.T) {
	var (
		bucket = "mybucket"
		prefix = "output/"
		dir    = "./test_files" + string(os.PathSeparator)
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-00.txt",
		fileMatcher{"test_files" + string(os.PathSeparator) + "test-00.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-01.txt",
		fileMatcher{"test_files" + string(os.PathSeparator) + "test-01.txt"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files/test-02.txt",
		fileMatcher{"test_files" + string(os.PathSeparator) + "test-02.txt"}, nil).Return(errors.New("done"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("-r %s %s:%s", dir, bucket, prefix)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "done" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestPutDirectoryFromDot(t *testing.T) {
	var (
		bucket = "mybucket"
		prefix = "output/"
		dir    = "."
	)
	prev, _ := filepath.Abs(".")
	target := filepath.Join(prev, "test_files2")
	os.Chdir(target)
	defer os.Chdir(prev)

	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, "output/test_files2/..dummy", fileMatcher{"..dummy"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files2/.dummy", fileMatcher{".dummy"}, nil).Return(nil)
	mock.EXPECT().UploadFile(bucket, "output/test_files2/dummy", fileMatcher{"dummy"}, nil).Return(errors.New("done"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("-r %s %s:%s", dir, bucket, prefix)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "done" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestPutDirectoryWithoutOption(t *testing.T) {
	var (
		bucket = "mybucket"
		key    = "test.txt"
		path   = "test_files" + string(os.PathSeparator)
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().UploadFile(bucket, key, fileMatcher{path}, nil).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs(fmt.Sprintf("test_files/ %s:%s", bucket, key)))
	if err.Error() != "\"test_files/\" is a directory (not uploaded)" {
		t.Errorf("Error message was not match. \"test_files/\" is a directory (not uploaded) != %v", err.Error())
	}
}

func TestPutBucketSelectRegion(t *testing.T) {
	var (
		mybucket = "mybucket"
		region   = "ap2"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(putCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().SelectRegionPutBucket(mybucket, region).Return(nil)
	c.cli = mock

	err := c.Run(parseArgs(fmt.Sprintf("-region=%s %s", region, mybucket)))
	if err != nil {
		t.Errorf("Error to execute put bucket wit region")
	}
}