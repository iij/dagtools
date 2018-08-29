package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type rmCommand struct {
	env       *env.Environment
	cli       client.StorageClient
	opts      *flag.FlagSet
	recursive bool
}

func (c *rmCommand) Description() string {
	return "delete a bucket or object[s]"
}

func (c *rmCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  rm [-r] <bucket>[:<file|dir>] ...

Options:
%v`, OptionUsage(c.opts))
}

func (c *rmCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("rm", flag.ExitOnError)
	opts.BoolVar(&c.recursive, "r", false, "recursively upload")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *rmCommand) removeBucket(bucket string) (err error) {
	if c.recursive {
		var (
			listing *client.ObjectListing
			res     *client.MultipleDeletionResult
		)
		if listing, err = c.cli.ListObjects(bucket, "", "", "", 1000); err != nil {
			return err
		}
		for {
			if listing == nil || len(listing.Summaries) < 1 {
				break
			}
			keys := make([]string, len(listing.Summaries))
			for i, o := range listing.Summaries {
				keys[i] = o.Key
			}
			if res, err = c.cli.DeleteMultipleObjects(bucket, keys, false); err != nil {
				return err
			}
			for _, o := range res.DeletedObjects {
				if c.env.Verbose {
					fmt.Printf("delete: %s/%s\n", bucket, o.Key)
				}
			}
			if res.HasErrors() {
				for _, e := range res.Errors {
					c.env.Logger.Println(e.String())
				}
			}
			if listing.IsTruncated {
				if listing, err = c.cli.NextListObjects(listing); err != nil {
					return err
				}
			} else {
				listing = nil
			}
		}
	}
	if c.env.Verbose {
		fmt.Printf("delete: %s\n", bucket)
	}
	err = c.cli.DeleteBucket(bucket)
	return err
}

func (c *rmCommand) removeObject(bucket string, prefix string) (num int, err error) {
	var (
		listing *client.ObjectListing
		res     *client.MultipleDeletionResult
	)
	if c.recursive {
		_prefix := strings.TrimRight(prefix, "*")
		if listing, err = c.cli.ListObjects(bucket, _prefix, "", "", 1000); err != nil {
			return
		}
		for {
			if listing == nil || len(listing.Summaries) < 1 {
				break
			}
			var keys []string
			for _, o := range listing.Summaries {
				d := _prefix
				if !strings.HasSuffix(prefix, "*") && !strings.HasSuffix(d, "/") {
					d += "/"
				}
				if o.Key != prefix && strings.Index(o.Key, d) != 0 {
					continue
				}
				keys = append(keys, o.Key)
			}
			res, err = c.cli.DeleteMultipleObjects(bucket, keys, false)
			if err != nil {
				return
			}
			num += len(res.DeletedObjects)
			for _, o := range res.DeletedObjects {
				if c.env.Verbose {
					fmt.Printf("delete: %s:%s\n", bucket, o.Key)
				}
			}
			if res.HasErrors() {
				for _, e := range res.Errors {
					c.env.Logger.Println(e.String())
				}
			}
			if listing != nil && listing.IsTruncated {
				listing, err = c.cli.NextListObjects(listing)
				if err != nil {
					return
				}
			} else {
				listing = nil
			}
		}
	} else {
		if strings.HasSuffix(prefix, "*") {
			_prefix := strings.TrimRight(prefix, "*")
			listing, err = c.cli.ListObjects(bucket, _prefix, "", "/", 1000)
			if err != nil {
				return
			}
			for {
				if listing == nil {
					break
				}
				keys := make([]string, len(listing.Summaries))
				for i, s := range listing.Summaries {
					keys[i] = s.Key
				}
				res, err = c.cli.DeleteMultipleObjects(bucket, keys, false)
				if err != nil {
					return
				}
				num += len(res.DeletedObjects)
				if c.env.Verbose {
					for _, o := range res.DeletedObjects {
						fmt.Printf("delete: %s:%s\n", bucket, o.Key)
					}
					for _, e := range res.Errors {
						fmt.Fprintf(os.Stderr, "[Error] %s\n", e.Error())
					}
				}
				if listing.IsTruncated {
					listing, err = c.cli.NextListObjects(listing)
					if err != nil {
						return
					}
				} else {
					listing = nil
				}
			}
		} else {
			err = c.cli.DeleteObject(bucket, prefix)
			if err != nil {
				return
			}
			num++
			if c.env.Verbose {
				fmt.Printf("delete: %s:%s\n", bucket, prefix)
			}
		}
	}
	return
}

func (c *rmCommand) Run(args []string) error {
	// initialize
	if len(args) == 0 {
		return ErrArgument
	}
	c.opts.Parse(args)
	argv := c.opts.Args()
	var (
		bucket = ""
		key    = ""
	)
	for _, arg := range argv {
		slice := strings.Split(arg, ":")
		bucket = slice[0]
		key = strings.Join(slice[1:], ":")
		if err := c.exec(bucket, key); err != nil {
			return err
		}
	}
	return nil
}

func (c *rmCommand) exec(bucket, key string) error {
	if key == "" {
		return c.removeBucket(bucket)
	}
	num, err := c.removeObject(bucket, key)
	if num == 0 && strings.HasPrefix(key, "/") {
		return c.exec(bucket, strings.TrimLeft(key, "/"))
	}
	return err
}

func init() {
	Commands.Register(new(rmCommand), "rm")
}
