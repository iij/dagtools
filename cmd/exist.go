package cmd

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type existCommand struct {
	env *env.Environment
	cli client.StorageClient
	opts		*flag.FlagSet
	printRegion bool
}

func (c *existCommand) Description() string {
	return "check to exist buckets/objects"
}

func (c *existCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  exist [-region] [<bucket>[:<key>] ...]

Options:
%s`, OptionUsage(c.opts))
}

func (c *existCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("exist", flag.ExitOnError)
	opts.BoolVar(&c.printRegion, "region", false, "output bucket region")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *existCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	var (
		bucket = ""
		key    = ""
	)
	argv := c.opts.Args()
	if len(argv) == 0 {
		return ErrArgument
	}
	if c.printRegion {
		for _, arg := range argv {
			xs := strings.Split(arg, ":")
			bucket = xs[0]
			var exist bool
			if len(xs) > 1 {
				key = strings.Join(xs[1:], ":")
			}
			exist,bucketLocation, err := c.exec(bucket, key)
			if err != nil {
				return err
			}
			if !exist {
				return fmt.Errorf("%s does not exist", arg)
			}
			if c.env.Verbose {
				fmt.Fprintf(os.Stdout, "%s does exist(%s)\n", arg, bucketLocation)
			}
		}
	} else {
		for _, arg := range argv {
			xs := strings.Split(arg, ":")
			bucket = xs[0]
			var exist bool
			if len(xs) > 1 {
				key = strings.Join(xs[1:], ":")
			}
			exist, _, err := c.exec(bucket, key)
			if err != nil {
				return err
			}
			if !exist {
				return fmt.Errorf("%s does not exist", arg)
			}
			if c.env.Verbose {
				fmt.Fprintf(os.Stdout, "%s does exist\n", arg)
			}
		}
	}
	return
}

func (c *existCommand) exec(bucket, key string) (exist bool, bucketLocation string, err error) {
	if key == "" {
		return c.cli.DoesBucketExist(bucket)
	}
	exist, bucketLocation, err = c.cli.DoesObjectExist(bucket, key)
	if (err == nil && !exist) && strings.HasPrefix(key, "/") {
		return c.exec(bucket, strings.TrimLeft(key, "/"))
	}
	return
}

func init() {
	Commands.Register(new(existCommand), "exist")
}
