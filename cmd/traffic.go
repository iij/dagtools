package cmd

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type trafficCommand struct {
	env           *env.Environment
	cli           client.StorageClient
	opts          *flag.FlagSet
	humanReadable bool
	backwardTo    int
}

func (c *trafficCommand) Description() string {
	return "display network traffics"
}

func (c *trafficCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  traffic [-h] [-b=N] [yyyyMMdd]

Options:
%s`, OptionUsage(c.opts))
}

func (c *trafficCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("traffic", flag.ExitOnError)
	opts.IntVar(&c.backwardTo, "b", -1, "dating back to the number of specified month")
	opts.BoolVar(&c.humanReadable, "h", false, "Human-readable output. Use unit suffix(B, KB, MB...) for sizes")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *trafficCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	argv := c.opts.Args()
	if len(argv) == 1 && argv[0] != "" {
		traffic, err := c.cli.GetNetworkTraffic(argv[0])
		if err == nil {
			c.printHeader()
			c.printTraffic(traffic)
		}
		return err
	}
	if c.backwardTo >= 0 {
		result, err := c.cli.ListNetworkTraffics(c.backwardTo)
		if err == nil {
			c.printHeader()
			traffics := result.DownTraffics
			for i := range traffics {
				c.printTraffic(traffics[i])
			}
		}
		return err
	}
	return ErrArgument
}

func (c *trafficCommand) printHeader() {
	fmt.Printf("%10s  %13s\n", "date", "down_traffic")
}

func (c *trafficCommand) printTraffic(traffic *client.DownTraffic) {
	if traffic == nil {
		return
	}
	var (
		date   string
		amount string
	)
	d := traffic.ChargeDate
	date = fmt.Sprintf("%s-%s-%s", d[0:4], d[4:6], d[6:])

	if c.humanReadable {
		amount = HumanReadableBytes(uint64(traffic.Amount))
	} else {
		amount = strconv.FormatInt(traffic.Amount, 10)
	}
	fmt.Printf("%10s  %13s\n", date, amount)
}

func init() {
	Commands.Register(new(trafficCommand), "traffic")
}
