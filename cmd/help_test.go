package cmd

import (
	"strings"
	"testing"

	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
)

func TestHelpUsage(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(helpCommand)
	c.Init(&e)
	usage := c.Usage()
	if !strings.HasPrefix(usage, "Command Usage:") {
		t.Errorf("Failed to get a cat command usage. usage: %q", usage)
	}
}

func TestHelpCommand(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(helpCommand)
	c.Init(&e)
	err := c.Run(parseArgs("cat"))
	if err != nil {
		t.Error("Failed to run a help command.", err)
	}
}

func TestHelpUnknownCommand(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	e := env.Environment{Config: config}
	e.Init()
	c := new(helpCommand)
	c.Init(&e)
	err := c.Run(parseArgs("unknown"))
	if err == nil {
		t.Error("Failed to get an error that command is unknown.", err)
	}
}
