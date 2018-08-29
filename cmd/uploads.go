package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

var uploadsSubCommands = map[string]Command{}

type uploadsCommand struct {
	env  *env.Environment
	cli  client.StorageClient
	opts *flag.FlagSet
}

func (c *uploadsCommand) Description() string {
	return "manage multipart-upload[s]"
}

func (c *uploadsCommand) Usage() string {
	return `Command Usage:
  uploads help [ls|rm]
  uploads ls [<bucket>[:<prefix>]]
  uploads rm <bucket>:<key>[:<uploadId>] ...`
}

func (c *uploadsCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("uploads", flag.ExitOnError)
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	for _, cmd := range uploadsSubCommands {
		cmd.Init(env)
	}
	return
}

func (c *uploadsCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		c.Usage()
		return ErrArgument
	}
	switch args[0] {
	case "ls", "rm":
		if cmd, err := getCmd(args[0]); err == nil {
			return cmd.Run(args[1:])
		}
	case "help":
		if len(args) > 1 {
			if cmd, err := getCmd(args[1]); err == nil {
				fmt.Println(cmd.Usage())
				return nil
			}
		} else {
			fmt.Println(c.Usage())
			return nil
		}
	}
	return ErrArgument
}

func getCmd(name string) (Command, error) {
	return uploadsSubCommands[name], nil
}

func init() {
	Commands.Register(new(uploadsCommand), "uploads")
}
