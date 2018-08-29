package cmd

import (
	"flag"
	"fmt"
	"os"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type spaceCommand struct {
	env           *env.Environment
	cli           client.StorageClient
	opts          *flag.FlagSet
	humanReadable bool
}

func (c *spaceCommand) Description() string {
	return "display used storage space"
}

func (c *spaceCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  space [-h]

Options:
%s`, OptionUsage(c.opts))
}

func (c *spaceCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("space", flag.ExitOnError)
	opts.BoolVar(&c.humanReadable, "h", false, "Human-readable output. Use unit suffix(B, KB, MB...) for sizes")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *spaceCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	usage, err := c.cli.GetStorageSpace()
	if err != nil {
		return
	}
	fmt.Printf("%13s %13s\n", "total", "account")
	if c.humanReadable {
		fmt.Printf("%13s %13s\n",
			HumanReadableBytes(uint64(usage.AccountUsed)),
			HumanReadableBytes(uint64(usage.ContractUsed)))
	} else {
		fmt.Printf("%13v %13v\n", usage.AccountUsed, usage.ContractUsed)
	}
	return
}

func init() {
	Commands.Register(new(spaceCommand), "space")
}
