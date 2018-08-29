package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

var (
	policySubCommands = map[string]bool{
		"put": true,
		"rm":  true,
		"cat": true,
	}
)

type policyCommand struct {
	env      *env.Environment
	cli      client.StorageClient
	isPolicy bool
}

func (c *policyCommand) Description() string {
	return "manage a bucket policy (put, cat, rm)"
}

func (c *policyCommand) Usage() string {
	return `Command Usage:
  policy cat <bucket>
  policy rm <bucket>
  policy put <bucket> <file>
  policy put <bucket> < <file>`
}

func (c *policyCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, err = client.NewStorageClient(env)
	return
}

func (c *policyCommand) Run(args []string) (err error) {
	var (
		command = ""
		bucket  = ""
	)
	if len(args) < 2 {
		return ErrArgument
	}
	command = args[0]
	bucket = args[1]
	if !policySubCommands[command] {
		return fmt.Errorf("policy's sub-command not found: %q", command)
	}
	switch command {
	case "put":
		var in *os.File
		stat, _ := os.Stdin.Stat()
		if stat == nil || (stat.Mode()&os.ModeCharDevice) != 0 {
			if len(args) != 3 {
				return ErrArgument
			}
			in, err = os.Open(args[2])
			if err != nil {
				return err
			}
			defer in.Close()
		} else {
			in = os.Stdin
		}
		err = c.cli.PutBucketPolicy(bucket, in)
		return err
	case "cat":
		out := os.Stdout
		r, err := c.cli.GetBucketPolicy(bucket)
		if err != nil {
			return err
		}
		bin := bufio.NewReader(r)
		bout := bufio.NewWriter(out)
		_, err = bin.WriteTo(bout)
		if err != nil {
			return err
		}
		bout.Flush()
		fmt.Println()
		return nil
	case "rm":
		err = c.cli.DeleteBucketPolicy(bucket)
		return
	}
	return
}

func init() {
	Commands.Register(new(policyCommand), "policy")
}
