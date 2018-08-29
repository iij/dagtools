package cmd

import (
	"bufio"
	"os"
	"strings"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type catCommand struct {
	env *env.Environment
	cli client.StorageClient
}

func (c *catCommand) Description() string {
	return "get an object and print to standard output"
}

func (c *catCommand) Usage() string {
	return `Command Usage:
  cat <bucket>:<key>
`
}

func (c *catCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	return
}

func (c *catCommand) Run(args []string) (err error) {
	var (
		bucket = ""
		key    = ""
	)
	if len(args) != 1 {
		return ErrArgument
	}
	slice := strings.Split(args[0], ":")
	if len(slice) < 2 {
		return ErrArgument
	}
	bucket = slice[0]
	key = strings.Join(slice[1:], ":")
	if key == "" {
		return ErrArgument
	}
	return c.exec(bucket, key)
}

func (c *catCommand) exec(bucket, key string) (err error) {
	r, err := c.cli.GetObject(bucket, key)
	if err != nil {
		if strings.HasPrefix(key, "/") {
			return c.exec(bucket, strings.TrimLeft(key, "/"))
		}
		return err
	}
	in := bufio.NewReader(r)
	out := bufio.NewWriter(os.Stdout)
	in.WriteTo(out)
	out.Flush()
	return
}

func init() {
	Commands.Register(new(catCommand), "cat")
}
