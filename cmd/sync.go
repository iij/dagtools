package cmd

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type syncCommand struct {
	env     *env.Environment
	cli     client.StorageClient
	opts    *flag.FlagSet
	dryRun  bool
	verbose bool
}

func (c *syncCommand) Description() string {
	return "synchronize with objects on DAG storage and local files"
}

func (c *syncCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  sync [-v] [-n] <bucket>:[<key prefix>] <dir>
  sync [-v] [-n] <dir> <bucket>:[<key prefix>]

Options:
%v`, OptionUsage(c.opts))
}

func (c *syncCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("sync", flag.ExitOnError)
	opts.BoolVar(&c.dryRun, "n", false, "show what would have been transferred(dry-run)")
	opts.BoolVar(&c.verbose, "v", env.Verbose, "verbose mode")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *syncCommand) SyncLocalToDag(bucket string, prefix string, dir string) (err error) {
	if strings.HasPrefix(dir, "./") {
		dir = dir[2:]
	}
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if dir != "" && !strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir += string(os.PathSeparator)
	}
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			fd, err := os.Open(path)
			if err != nil {
				return err
			}
			defer fd.Close()
			fstat, err := fd.Stat()
			if err != nil {
				return err
			}
			key := path
			if strings.HasPrefix(path, dir) {
				key = path[len(dir):]
			}
			key = prefix + strings.TrimLeft(key, string(os.PathSeparator))
			key = strings.Replace(key, string(os.PathSeparator), "/", -1)
			o, err := c.cli.GetObjectMetadata(bucket, key)
			if err != nil {
				return err
			}
			if o != nil {
				lastModified := o.LastModified
				timestr := o.Metadata.GetUserMetadata("last_modified")
				if timestr != "" {
					t, _ := strconv.Atoi(timestr)
					lastModified = time.Unix(int64(t), 0)
				}
				size := o.Size
				if size < 0 {
					size = 0
				}
				if size == info.Size() && lastModified.Unix() == fstat.ModTime().Unix() {
					if c.env.Debug {
						c.env.Logger.Printf("no change. %s:%s = %s", bucket, key, path)
					}
					return nil
				}
			}
			if !c.dryRun {
				c.env.Logger.Printf("Uploading %q to %s:%s ...", path, bucket, key)
				metadata := new(client.ObjectMetadata)
				lastModified := strconv.Itoa(int(fstat.ModTime().Unix()))
				metadata.AddUserMetadata("last_modified", lastModified)
				if err := c.cli.UploadFile(bucket, key, fd, metadata); err != nil {
					c.env.Logger.Printf(fmt.Sprintf("Failed to upload %q. %s", path, err))
					fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
				} else if c.env.Verbose {
					fmt.Printf("put: %s -> %s:%s\n", path, bucket, key)
				}
			} else {
				if c.env.Verbose {
					fmt.Printf("put: %s -> %s:%s (dry-run)\n", path, bucket, key)
				}
			}
		}
		return nil
	})
}

func (c *syncCommand) SyncDagToLocal(bucket string, prefix string, dir string) (err error) {
	var listing *client.ObjectListing
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if !strings.HasSuffix(dir, "/") {
		dir += "/"
	}
	fd, err := os.Open(dir)
	if err != nil {
		return err
	}
	_, err = fd.Stat()
	if err != nil {
		return err
	}
	if !c.dryRun {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	if listing, err = c.cli.ListObjects(bucket, prefix, "", "", 1000); err != nil {
		return err
	}
	for {
		if listing == nil || len(listing.Summaries) < 1 {
			break
		}
		for _, o := range listing.Summaries {
			name := strings.Replace(o.Key, prefix, "", 1)
			target := strings.Replace(dir+name, string(os.PathSeparator), "/", -1)
			if name == "" {
				continue
			}
			c.WriteFile(&o, bucket, target)
		}
		if listing != nil && listing.IsTruncated {
			listing, err = c.cli.NextListObjects(listing)
			if err != nil {
				return err
			}
		} else {
			listing = nil
		}
	}
	return
}

func (c *syncCommand) WriteFile(o *client.ObjectSummary, bucket string, target string) (err error) {
	if c.dryRun {
		if c.env.Verbose {
			fmt.Printf("get: %s:%s -> %s (dry-run)\n", bucket, o.Key, target)
		}
		return
	}
	if strings.HasSuffix(o.Key, "/") {
		return
	}
	err = os.MkdirAll(path.Dir(target), 0755)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
		return
	}
	fd, err := os.Open(target)
	if err == nil {
		defer fd.Close()
		if fi, err := fd.Stat(); err == nil {
			if o.Size == fi.Size() {
				lastModified := o.LastModified
				if m, _ := c.cli.GetObjectMetadata(bucket, o.Key); m != nil {
					if timestr := m.Metadata.GetUserMetadata("last_modified"); timestr != "" {
						t, _ := strconv.Atoi(timestr)
						lastModified = time.Unix(int64(t), 0)
					}
					if lastModified.Unix() == fi.ModTime().Unix() {
						if c.env.Debug {
							c.env.Logger.Printf("no change. %s = %s:%s", target, bucket, o.Key)
						}
						return nil
					}
				}
			}
		}
	} else {
		if c.env.Debug {
			c.env.Logger.Print(err)
		}
	}
	r, err := c.cli.GetObject(bucket, o.Key)
	if err != nil {
		c.env.Logger.Printf("Failed to get object: %s/%s. %s", bucket, o.Key, err)
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
		return
	}
	defer r.Close()
	file, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
		return
	}
	defer file.Close()
	br := bufio.NewReader(r)
	bw := bufio.NewWriter(file)
	defer func(o *client.ObjectSummary) {
		bw.Flush()
		os.Chtimes(target, o.LastModified, o.LastModified)
	}(o)
	if c.env.Verbose {
		fmt.Printf("get: %s:%s -> %s\n", bucket, o.Key, target)
	}
	if _, err := br.WriteTo(bw); err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
		c.env.Logger.Printf("Failed to write file: %s. %s", target, err)
	}
	return
}

func (c *syncCommand) Run(args []string) (err error) {
	var (
		bucket  string
		prefix  string
		target  string
		toLocal bool
	)
	c.opts.Parse(args)
	argv := c.opts.Args()
	for _, arg := range argv {
		if strings.Contains(arg, ":") {
			slice := strings.Split(arg, ":")
			bucket = slice[0]
			prefix = strings.Join(slice[1:], ":")
		} else {
			toLocal = bucket != ""
			target = arg
		}
	}
	if strings.HasPrefix(prefix, "/") {
		return errors.New("object key must not include the slash(/) at the beginning of the value")
	}
	if bucket == "" || target == "" {
		return ErrArgument
	}

	// sync local directory with remote bucket/folder
	if toLocal {
		return c.SyncDagToLocal(bucket, prefix, target)
	}
	return c.SyncLocalToDag(bucket, prefix, target)
}

func init() {
	c := new(syncCommand)
	c.dryRun = false
	Commands.Register(c, "sync")
}
