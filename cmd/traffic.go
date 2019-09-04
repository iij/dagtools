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
	total 		  bool
	region		  string
}

func (c *trafficCommand) Description() string {
	return "display network traffics"
}

func (c *trafficCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  traffic [-h] [-t] [-region=ap1(or ap2)] [-b=N] [yyyyMMdd]

Options:
%s`, OptionUsage(c.opts))
}

func (c *trafficCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	opts := flag.NewFlagSet("traffic", flag.ExitOnError)
	opts.IntVar(&c.backwardTo, "b", -1, "dating back to the number of specified month")
	opts.BoolVar(&c.humanReadable, "h", false, "Human-readable output. Use unit suffix(B, KB, MB...) for sizes")
	opts.BoolVar(&c.total, "t", false, "Total traffic of all regions.")
	opts.StringVar(&c.region,"region", "", "Identifier of region")
	opts.Usage = func() {
		fmt.Fprintln(os.Stdout, c.Usage())
	}
	c.opts = opts
	return
}

func (c *trafficCommand) Run(args []string) (err error) {
	c.opts.Parse(args)
	argv := c.opts.Args()
	if c.total {
		region, err := c.cli.GetRegions()
		if err != nil {
			return err
		}
		if len(argv) == 1 && argv[0] != "" {
			var totalTraffic client.DownTraffic
			totalTraffic.ChargeDate = argv[0]
				for _, r := range region.Regions {
					traffic, err := c.cli.GetNetworkTraffic(argv[0], r.Name)
					if err != nil {
						return err
					}
				totalTraffic.Amount += traffic.Amount
			}
			if err == nil {
				c.printHeader()
				c.printTraffic(&totalTraffic)
			}
			return err
		}
		if c.backwardTo >= 0 {
			var totalResult []client.DownTraffic
			var result		 *client.ListTrafficResult
			for i, r := range region.Regions {
				result, err = c.cli.ListNetworkTraffics(c.backwardTo, r.Name)
				if err != nil {
					return err
				}
				// にるぽ対策だけど常套手段がありそう
				if i == 0 {
					for i := range result.DownTraffics {
						totalResult = append(totalResult,*result.DownTraffics[i])
						totalResult[i].Amount = 0
					}
				}
				for i := range result.DownTraffics {
					totalResult[i].Amount += result.DownTraffics[i].Amount
					fmt.Println(result.DownTraffics[i].Amount)
					fmt.Println(totalResult[i].Amount)
				}
			}
			if err == nil {
				c.printHeader()
				traffics := totalResult
				for i := range traffics {
					c.printTraffic(&traffics[i])
				}
			}
			return err
		}
	} else {
		if len(argv) == 1 && argv[0] != "" {
			traffic, err := c.cli.GetNetworkTraffic(argv[0], c.region)
			if err == nil {
				c.printHeader()
				c.printTraffic(traffic)
			}
			return err
		}
		if c.backwardTo >= 0 {
			result, err := c.cli.ListNetworkTraffics(c.backwardTo, c.region)
			if err == nil {
				c.printHeader()
				traffics := result.DownTraffics
				for i := range traffics {
					c.printTraffic(traffics[i])
				}
			}
			return err
		}
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
