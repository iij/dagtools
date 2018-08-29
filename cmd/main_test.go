package cmd

import (
	"strings"
	"testing"

	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
)

type dummyCommand struct {
	callDescription bool
	callUsage       bool
	callInit        bool
	callRun         bool
	args            []string
}

func (cmd *dummyCommand) Description() string {
	cmd.callDescription = true
	return "description"
}

func (cmd *dummyCommand) Usage() string {
	cmd.callUsage = true
	return "usage"
}

func (cmd *dummyCommand) Init(env *env.Environment) error {
	cmd.callInit = true
	return nil
}

func (cmd *dummyCommand) Run(args []string) error {
	cmd.args = args
	cmd.callRun = true
	return nil
}

var _dummyCmd = new(dummyCommand)

func init() {
	Register(_dummyCmd, "dummy")
}

func prepare() {
	_dummyCmd.callDescription = false
	_dummyCmd.callUsage = false
	_dummyCmd.callInit = false
	_dummyCmd.callRun = false
}

func TestUnknownCommand(t *testing.T) {
	prepare()
	e := env.Environment{}
	returnCode := Run(&e, "unknown", strings.Split("", " "))
	if returnCode != 1 {
		t.Errorf("return code did not match. 1 != %v", returnCode)
	}
}

func TestRunSubCommand(t *testing.T) {
	prepare()
	config := &ini.Config{
		Filename: "dummy.ini",
		Sections: make(map[string]ini.Section),
	}
	config.Set("dagtools", "verbose", "false")
	e := env.Environment{
		Verbose: false,
		Debug:   false,
		Config:  config,
	}
	e.Init()
	returnCode := Run(&e, "dummy", strings.Split("foo bar", " "))
	if returnCode != 0 {
		t.Errorf("return code did not match. 0 != %v", returnCode)
	}
	if !_dummyCmd.callInit {
		t.Errorf("dummyCommand::Init did not call.")
	}
	if !_dummyCmd.callRun {
		t.Errorf("dummyCommand::Run did not call.")
	}
	if len(_dummyCmd.args) != 2 {
		t.Errorf("unmatched length of arguments. 2 != %v", len(_dummyCmd.args))
	}
	if _dummyCmd.args[0] != "foo" || _dummyCmd.args[1] != "bar" {
		t.Errorf("Invalid command arguments. %v", _dummyCmd.args)
	}
}
