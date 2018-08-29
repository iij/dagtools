package cmd

import (
	"fmt"
	"os"

	"github.com/iij/dagtools/env"
)

type helpCommand struct {
	env *env.Environment
}

func (c *helpCommand) Description() string {
	return "print a command usage"
}

func (c *helpCommand) Usage() string {
	return `Command Usage:
  help <command>
`
}

func (c *helpCommand) Init(env *env.Environment) (err error) {
	c.env = env
	return
}

func (c *helpCommand) Run(args []string) (err error) {
	if len(args) == 0 {
		return ErrArgument
	}
	var (
		name string
		_c   Command
	)
	name = args[0]
	cs := Commands.All()
	if _c = cs[name]; _c == nil {
		return fmt.Errorf("command not found: %q", name)
	}
	_c.Init(c.env)
	fmt.Fprintf(os.Stdout, "%s - %s\n\n", name, _c.Description())
	fmt.Fprintln(os.Stdout, _c.Usage())
	return
}

func init() {
	Commands.Register(new(helpCommand), "help")
}
