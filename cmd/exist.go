package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type existCommand struct {
	env *env.Environment
	cli client.StorageClient
}

func (c *existCommand) Description() string {
	return "check to exist buckets/objects"
}

func (c *existCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  exist [<bucket>[:<key>] ...]`)
}

func (c *existCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	return
}

func (c *existCommand) Run(args []string) (err error) {
	var (
		bucket = ""
		key    = ""
	)
	if len(args) == 0 {
		return ErrArgument
	}
	for _, arg := range args {
		xs := strings.Split(arg, ":")
		bucket = xs[0]
		var exist bool
		if len(xs) > 1 {
			key = strings.Join(xs[1:], ":")
		}
		exist, err := c.exec(bucket, key)
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
	return
}

func (c *existCommand) exec(bucket, key string) (exist bool, err error) {
	if key == "" {
		return c.cli.DoesBucketExist(bucket)
	}
	exist, err = c.cli.DoesObjectExist(bucket, key)
	if (err == nil && !exist) && strings.HasPrefix(key, "/") {
		return c.exec(bucket, strings.TrimLeft(key, "/"))
	}
	return
}

func init() {
	Commands.Register(new(existCommand), "exist")
}
