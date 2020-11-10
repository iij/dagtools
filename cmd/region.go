package cmd

import (
	"fmt"
	"github.com/iij/dagtools/client"
	"github.com/iij/dagtools/env"
)

type regionCommand struct {
	env           *env.Environment
	cli           client.StorageClient
}

func (c *regionCommand) Description() string {
	return "display all regions info"
}

func (c *regionCommand) Usage() string {
	return fmt.Sprintf(`Command Usage:
  region`)
}

func (c *regionCommand) Init(env *env.Environment) (err error) {
	c.env = env
	c.cli, _ = client.NewStorageClient(env)
	return
}

func (c *regionCommand) Run(args []string) (err error) {
	if len(args) != 0 {
		return ErrArgument
	}
	regions, err := c.cli.GetRegions()
	if err != nil {
		return err
	}
	if len(regions.Regions) != 0 {
		fmt.Printf("%s\t %s\n", "name", "endpoint")
		for _, r := range regions.Regions {
			fmt.Printf("%s\t %s\n", r.Name, r.Endpoint)
		}
	}
	return
}

func init() {
	Commands.Register(new(regionCommand), "region")
}
