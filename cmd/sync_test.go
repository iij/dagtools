package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
	"github.com/golang/mock/gomock"
)

func TestSyncUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestSyncDagToLocal(t *testing.T) {
	var (
		bucket = "mybucket"
		from   = "dummy/"
		to     = "test_files/"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)
	mock.EXPECT().ListObjects(bucket, from, "", "", 1000).Return(nil, errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("%s:%s %s", bucket, from, to)))
	if err == nil {
		t.Error("Failed to get an error.", err)
	}
	if err.Error() != "dummy" {
		t.Errorf("Error message was not match. dummy != %v", err.Error())
	}
}

func TestSyncLocalToDag(t *testing.T) {
	var (
		bucket = "mybucket"
		from   = "test_files" + string(os.PathSeparator)
		to     = "dummy/"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)

	// new
	fd00, _ := os.Open(from + "test-00.txt")
	m00 := new(client.ObjectMetadata)
	fstat00, _ := fd00.Stat()
	m00.AddUserMetadata("last_modified", strconv.Itoa(int(fstat00.ModTime().Unix())))
	mock.EXPECT().GetObjectMetadata(bucket, to+"test-00.txt").Return(nil, nil)
	mock.EXPECT().UploadFile(bucket, to+"test-00.txt", fileMatcher{fd00.Name()}, metadataMatcher{m00}).Return(errors.New("dummy"))

	// no change
	fd01, _ := os.Open(from + "test-01.txt")
	fstat01, _ := fd01.Stat()
	m01 := new(client.ObjectMetadata)
	m01.AddUserMetadata("last_modified", strconv.Itoa(int(fstat01.ModTime().Unix())))
	o1 := new(client.Object)
	o1.Size = fstat01.Size()
	o1.LastModified = time.Now() // dummy
	o1.Metadata = m01
	mock.EXPECT().GetObjectMetadata(bucket, to+"test-01.txt").Return(o1, nil)

	// modified
	fd02, _ := os.Open(from + "test-02.txt")
	m02 := new(client.ObjectMetadata)
	fstat02, _ := fd02.Stat()
	o2 := new(client.Object)
	o2.Size = 1024
	o2.Metadata = new(client.ObjectMetadata)
	o2.LastModified = time.Unix(0, 0)
	m02.AddUserMetadata("last_modified", strconv.Itoa(int(fstat02.ModTime().Unix())))
	mock.EXPECT().GetObjectMetadata(bucket, to+"test-02.txt").Return(o2, nil)
	mock.EXPECT().UploadFile(bucket, to+"test-02.txt", fileMatcher{fd02.Name()}, metadataMatcher{m02}).Return(errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("%s %s:%s", from, bucket, to)))
	if err != nil {
		t.Error("unknown error", err)
	}
}

func TestSyncCurrentDirectoryToDag(t *testing.T) {
	var (
		bucket = "mybucket"
		from   = "."
		to     = "dummy/"
	)
	prev, _ := filepath.Abs(".")
	target := filepath.Join(prev, "test_files2")
	os.Chdir(target)

	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)
	ctrl := gomock.NewController(t)
	mock := client.NewMockStorageClient(ctrl)

	fd00, _ := os.Open("..dummy")
	m00 := new(client.ObjectMetadata)
	fstat00, _ := fd00.Stat()
	m00.AddUserMetadata("last_modified", strconv.Itoa(int(fstat00.ModTime().Unix())))
	mock.EXPECT().GetObjectMetadata(bucket, to+"..dummy").Return(nil, nil)
	mock.EXPECT().UploadFile(bucket, to+"..dummy", fileMatcher{fd00.Name()}, metadataMatcher{m00}).Return(errors.New("dummy"))

	fd01, _ := os.Open(".dummy")
	m01 := new(client.ObjectMetadata)
	fstat01, _ := fd01.Stat()
	m01.AddUserMetadata("last_modified", strconv.Itoa(int(fstat01.ModTime().Unix())))
	mock.EXPECT().GetObjectMetadata(bucket, to+".dummy").Return(nil, nil)
	mock.EXPECT().UploadFile(bucket, to+".dummy", fileMatcher{fd01.Name()}, metadataMatcher{m01}).Return(errors.New("dummy"))

	fd02, _ := os.Open("dummy")
	m02 := new(client.ObjectMetadata)
	fstat02, _ := fd02.Stat()
	m02.AddUserMetadata("last_modified", strconv.Itoa(int(fstat02.ModTime().Unix())))
	mock.EXPECT().GetObjectMetadata(bucket, to+"dummy").Return(nil, nil)
	mock.EXPECT().UploadFile(bucket, to+"dummy", fileMatcher{fd02.Name()}, metadataMatcher{m02}).Return(errors.New("dummy"))
	c.cli = mock
	err := c.Run(parseArgs(fmt.Sprintf("%s %s:%s", from, bucket, to)))
	if err != nil {
		t.Error("unknown error", err)
	}
	os.Chdir(prev)
}

func TestSyncNotExistDir(t *testing.T) {
	var (
		bucket = "mybucket"
		from   = "nosuchdir" + string(os.PathSeparator)
		to     = "dummy/"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)

	err := c.Run(parseArgs(fmt.Sprintf("%s %s:%s", from, bucket, to)))
	if err.Error() != "lstat nosuchdir/: no such file or directory" {
		if err.Error() != "GetFileAttributesEx nosuchdir\\: The system cannot find the file specified." {
			t.Error("This directory is not exists.", err)
		}
	}
}

func TestSyncNotDirLocalToDag(t *testing.T) {
	var (
		bucket = "mybucket"
		from   = "test_files" + string(os.PathSeparator) + "test-00.txt"
		to     = "dummy/"
	)
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(syncCommand)
	c.Init(&e)

	err := c.Run(parseArgs(fmt.Sprintf("%s %s:%s", from, bucket, to)))
	if err.Error() != "lstat test_files/test-00.txt/: not a directory" {
		if err.Error() != "GetFileAttributesEx test_files\\test-00.txt\\: The filename, directory name, or volume label syntax is incorrect." {
			t.Error("This Object is not a directory.", err)
		}
	}
}

type metadataMatcher struct {
	x *client.ObjectMetadata
}

func (m metadataMatcher) Matches(x interface{}) bool {
	_x := x.(*client.ObjectMetadata)
	return _x.String() == m.x.String()
}

func (m metadataMatcher) String() string {
	return "is metadata"
}
