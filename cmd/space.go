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
	region		  string
	total		  bool
}

func (c *spaceCommand) Description() string {
	return "display used storage space"
}

func (c *spaceCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  space [-h] [-t] [-region]

Options:
%s`, OptionUsage(c.opts))
}

func (c *spaceCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("space", flag.ExitOnError)
	opts.BoolVar(&c.humanReadable, "h", false, "Human-readable output. Use unit suffix(B, KB, MB...) for sizes")
	opts.StringVar(&c.region, "region", "", "Identifier of region to print")
	opts.BoolVar(&c.total, "t", false, "Print storage space of all region")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *spaceCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	if c.total {
		regions, err := c.cli.GetRegions()
		if err != nil {
			return err
		}
		var contractTotal, accountTotal int64
		for _, r := range regions.Regions {
			space, err := c.cli.GetStorageSpace(r.Name)
			if err != nil {
				return err
			}
			accountTotal += space.AccountUsed
			contractTotal += space.ContractUsed
		}
		fmt.Printf("%13s %13s\n", "contract", "account")
		if c.humanReadable {
			fmt.Printf("%13s %13s\n",
				HumanReadableBytes(uint64(contractTotal)),
				HumanReadableBytes(uint64(accountTotal)))
		} else {
			fmt.Printf("%13v %13v\n", contractTotal, accountTotal)
		}
		return nil
	}
	usage, err := c.cli.GetStorageSpace(c.region)
	if err != nil {
		return
	}
	fmt.Printf("%13s %13s\n", "total", "account")
	if c.humanReadable {
		fmt.Printf("%13s %13s\n",
			HumanReadableBytes(uint64(usage.ContractUsed)),
			HumanReadableBytes(uint64(usage.AccountUsed)))
	} else {
		fmt.Printf("%13v %13v\n", usage.ContractUsed, usage.AccountUsed)
	}
	return
}

func init() {
	Commands.Register(new(spaceCommand), "space")
}
