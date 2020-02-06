package cmd

import (
	"flag"
	"fmt"
	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
	"os"
)

type cprCommand struct {
	env   *env.Environment
	cli   client.StorageClient
	opts  *flag.FlagSet
	force bool
}

func (c *cprCommand) Description() string {
	return "copy all object to other bucket"
}

func (c *cprCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  cpr <bucket> <bucket>
  cpr -f <bucket> <bucket>

Options:
%s`, OptionUsage(c.opts))
}

func (c *cprCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("cpr", flag.ExitOnError)
	opts.BoolVar(&c.force, "f", false, "ignore modifies")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *cprCommand) Run(args []string) (err error) {
	var (
		sourceBucket = ""
		//destBucket = ""
	)
	if len(args) == 0 {
		return ErrArgument
	}
	c.opts.Parse(args)
	argv := c.opts.Args()
	if len(argv) != 2 {
		return ErrArgument
	}
	sourceBucket = argv[0]
	//destBucket = argv[1]
	_, err = c.cli.ListObjects(sourceBucket, "", "", "", 1000)
	if err != nil {
		return err
	}
	return
}

func (c *cprCommand) execPutObjectCopy(listing *client.ObjectListing, destBucket string) (int, error) {
	exist, _, err := c.cli.DoesBucketExist(destBucket)

	if err != nil || exist == false {
		return -1, err
	}
	if c.force {
		for _, o := range listing.Summaries {
			err = c.cli.PutObjectCopy(listing.Name, o.Key, destBucket, o.Key, "hoge")
			if err != nil {
				fmt.Printf("Fail to PUT object copy %s:%s\n", listing.Name, o.Key)
			}
		}
	} else {
		for _, o := range listing.Summaries {
			exist, _, err = c.cli.DoesObjectExist(destBucket, o.Key)
			if err != nil {
				return -1, err
			}
			if exist == true {
			}
		}
	}
	return 0, nil
}

func init() {
	Commands.Register(new(cprCommand), "cpr")
}
