package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type getCommand struct {
	env       *env.Environment
	cli       client.StorageClient
	opts      *flag.FlagSet
	recursive bool
}

func (c *getCommand) Description() string {
	return "get object[s] and write to file[s]"
}

func (c *getCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  get <bucket>:<key>
  get <bucket>:<key> <file>
  get -r <bucket>:<prefix>
  get -r <bucket>:<prefix> <dir>/
  get -r <bucket>:<prefix> <dir>/<dirname>

Options:
%s`, OptionUsage(c.opts))
}

func (c *getCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("get", flag.ExitOnError)
	opts.BoolVar(&c.recursive, "r", false, "recursively download")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *getCommand) Run(args []string) (err error) {
	var (
		bucket = ""
		key    = ""
		target = ""
	)
	// parse the sub-command line arguments
	c.opts.Parse(args)
	argv := c.opts.Args()

	// check specified args
	if len(argv) < 1 || 2 < len(argv) {
		return ErrArgument
	}
	slice := strings.Split(argv[0], ":")
	if len(slice) < 2 {
		return ErrArgument
	}
	bucket = slice[0]
	key = strings.Join(slice[1:], ":")
	if len(argv) >= 2 {
		target = argv[1]
	} else {
		target = path.Base(key)
	}
	return c.exec(bucket, key, target)
}

func (c *getCommand) exec(bucket, key, target string) (err error) {
	if c.recursive {
		err = c.getObjectRecursively(bucket, key, target)
	} else {
		err = c.getObject(bucket, key, target)
	}
	if err != nil {
		if strings.HasPrefix(key, "/") {
			return c.exec(bucket, key, target)
		}
		return err
	}
	return
}

func (c *getCommand) getObject(bucket string, key string, target string) (err error) {
	if key == "" {
		return ErrArgument
	}
	if target == "" {
		_names := strings.Split(key, "/")
		target = _names[len(_names)-1]
	}
	in, err := c.cli.GetObject(bucket, key)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer out.Close()
	bin := bufio.NewReader(in)
	bout := bufio.NewWriter(out)
	_, err = bin.WriteTo(bout)
	if err != nil {
		return err
	}
	err = bout.Flush()
	if err != nil {
		return err
	}
	return
}

func (c *getCommand) getObjectRecursively(bucket string, prefix string, dir string) (err error) {
	var listing *client.ObjectListing
	if strings.HasSuffix(dir, string(os.PathSeparator)) {
		dir = path.Join(dir, path.Base(prefix))
	}
	// check the target directory exists
	parent := path.Dir(dir)
	if _, err = os.Stat(parent); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("[Error] directory %s does not exist\n", parent)
		}
		return err
	}
	if listing, err = c.cli.ListObjects(bucket, prefix, "", "", 1000); err != nil {
		return err
	}
	for {
		if listing == nil || len(listing.Summaries) < 1 {
			break
		}
		for _, o := range listing.Summaries {
			if prefix != o.Key && !strings.HasSuffix(prefix, "/") && !strings.HasPrefix(o.Key, prefix+"/") {
				continue
			}
			name := strings.Replace(o.Key, prefix, "", 1)
			target := strings.Replace(path.Join(dir, name), string(os.PathSeparator), "/", -1)
			if err = os.MkdirAll(path.Dir(target), 0755); err != nil {
				fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
				continue
			}
			if strings.HasSuffix(name, "/") {
				continue
			}
			c.writeFile(bucket, o, target)
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

func (c *getCommand) writeFile(bucket string, o client.ObjectSummary, target string) (err error) {
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
	defer bw.Flush()
	if c.env.Verbose {
		fmt.Printf("get: %s:%s -> %s\n", bucket, o.Key, target)
	}
	if _, err := br.WriteTo(bw); err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err)
		c.env.Logger.Printf("Failed to write file: %s. %s", target, err)
	}
	return nil
}

func init() {
	Commands.Register(new(getCommand), "get")
}
