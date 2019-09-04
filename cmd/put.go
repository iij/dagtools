package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"errors"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type putCommand struct {
	env       *env.Environment
	cli       client.StorageClient
	opts      *flag.FlagSet
	recursive bool
	uploadId  string
	putRegion string
}

func (c *putCommand) Description() string {
	return "put a bucket or object[s]"
}

func (c *putCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  put <bucket>
  put <file> <bucket>[:<key>]
  put <file1> [<file2>...] <bucket>:<prefix>/
  put -r <dir> <bucket>:<prefix>[/]
  put -upload-id=<upload-id> <file> <bucket>[:<key>]
  put -region=<region=ap1(or ap2)> <file> <bucket>[:<key>]
  put <bucket>:<key> < <file>

Options:
%s`, OptionUsage(c.opts))
}

func (c *putCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("put", flag.ExitOnError)
	opts.BoolVar(&c.recursive, "r", false, "recursively upload")
	opts.StringVar(&c.uploadId, "upload-id", "", "identifier of multipart upload")
	opts.StringVar(&c.putRegion, "region", "", "identifier of region to put")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *putCommand) Run(args []string) (err error) {
	var (
		bucket = ""
		key    = ""
	)
	if len(args) == 0 {
		return ErrArgument
	}
	c.opts.Parse(args)
	argv := c.opts.Args()

	// Target resource: "{Bucket}:{Key}"
	slice := strings.Split(argv[len(argv)-1], ":")
	if len(slice) < 1 {
		return ErrArgument
	}
	bucket = slice[0]
	key = strings.Join(slice[1:], ":")
	// Object key does not start with a slash(/)
	if strings.HasPrefix(key, "/") {
		return errors.New("object key must not include the slash(/) at the beginning of the value")
	}
	// PUT Object from Standard Input
	stat, _ := os.Stdin.Stat()
	if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
		r := bufio.NewReader(os.Stdin)
		if _, err = r.Peek(1); err == nil {
			if key == "" {
				return ErrArgument
			}
			return c.cli.Upload(bucket, key, r, nil)
		}
	}
	// PUT Bucket
	if len(argv) < 2 && key == "" {
		if c.putRegion != "" {
			err = c.cli.SelectRegionPutBucket(bucket, c.putRegion)
		} else {
			err = c.cli.PutBucket(bucket)
		}
		return err
	}
	// PUT Object(s)
	// check specified file
	if len(argv) > 2 && key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	for _, path := range argv[:len(argv)-1] {
		if err = c.putFileOrDirectory(path, bucket, key); err != nil {
			return err
		}
	}
	return
}

func (c *putCommand) putFileOrDirectory(root string, bucket string, key string) (err error) {
	fd, err := os.Open(root)
	if err != nil {
		return err
	}
	defer fd.Close()
	fstat, err := fd.Stat()
	if err != nil {
		return err
	}
	switch mode := fstat.Mode(); {
	case mode.IsRegular():
		target := key
		if strings.HasSuffix(key, "/") || key == "" {
			target += filepath.Base(root)
		}
		if c.env.Verbose {
			fmt.Fprintf(os.Stdout, "put: %s -> %s:%s\n", root, bucket, target)
		}
		if c.uploadId != "" {
			if err = c.cli.ResumeUploadFile(bucket, target, c.uploadId, fd, nil); err != nil {
				return err
			}
		} else {
			if err = c.cli.UploadFile(bucket, target, fd, nil); err != nil {
				return err
			}
		}
	case mode.IsDir():
		if c.recursive {
			err := filepath.Walk(root,
				func(path string, info os.FileInfo, err error) error {
					if info.IsDir() {
						return nil
					}
					_root := filepath.Clean(root)
					if strings.HasPrefix(_root, "."+string(os.PathSeparator)) {
						_root = _root[2:]
						if _root == "" {
							_root = "."
						}
					}
					fd, err := os.Open(path)
					if err != nil {
						return err
					}
					defer fd.Close()
					target := key
					_path := path
					if _root != "." && strings.HasPrefix(path, _root) {
						_path = path[len(_root):]
					}
					if strings.HasPrefix(_path, string(os.PathSeparator)) {
						_path = strings.TrimLeft(_path, string(os.PathSeparator))
					}
					if strings.HasSuffix(key, "/") || key == "" {
						if _absRoot, err := filepath.Abs(_root); err == nil {
							target += filepath.Base(_absRoot) + "/" + _path
						}
					} else {
						target += "/" + _path
					}
					target = strings.Replace(target, string(os.PathSeparator), "/", -1)
					if c.env.Verbose {
						fmt.Fprintf(os.Stdout, "put: %s -> %s:%s\n", path, bucket, target)
					}
					if err = c.cli.UploadFile(bucket, target, fd, nil); err != nil {
						return err
					}
					return nil
				})
			if err != nil {
				return err
			}
		} else {
			return fmt.Errorf("%q is a directory (not uploaded)", root)
		}
	}
	return
}

func init() {
	Commands.Register(new(putCommand), "put")
}
