package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/iij/dagtools/cmd"
	_ "github.com/iij/dagtools/cmd"
	"github.com/iij/dagtools/env"
	"github.com/iij/dagtools/ini"
)

var (
	commandLine *flag.FlagSet
)

// Usage prints a command usage of the dagtools.
func Usage(out *os.File) {
	fmt.Fprintln(out, "Usage:\n  dagtools [-h] [-d] [-v] [-f <config file>] <command> [<args>]\n\nOptions:")
	commandLine.PrintDefaults()
	fmt.Fprintln(out, "\nCommands:")
	for name, _cmd := range cmd.Commands.All() {
		fmt.Fprintf(out, "%12s: %s\n", name, _cmd.Description())
	}
	fmt.Fprintln(out, "")
}

func main() {
	var (
		configFile         string
		config             ini.Config
		err                error
		defaultConfigFiles = [...]string{"dagtools.ini", "/etc/dagtools.ini"}
		usage              = false
		verbose            = false
		version            = false
		debug              = false
	)
	commandLine = flag.NewFlagSet("dagtools", flag.ExitOnError)
	commandLine.Usage = func() {
		Usage(os.Stderr)
	}
	commandLine.BoolVar(&usage, "h", false, "print a help message and exit")
	commandLine.BoolVar(&verbose, "v", false, "verbose mode")
	commandLine.BoolVar(&version, "version", false, "show version")
	commandLine.BoolVar(&debug, "d", false, "debug mode")
	commandLine.StringVar(&configFile, "f", "", "specify an alternate configuration file (default: ./dagtools.ini or /etc/dagtools.ini)")
	commandLine.Parse(os.Args[1:])
	args := commandLine.Args()

	if version {
		fmt.Fprintf(os.Stdout, "dagtools version %v\n", env.Version)
		os.Exit(0)
	}
	if usage {
		Usage(os.Stdout)
		os.Exit(0)
	}
	if len(args) < 1 {
		Usage(os.Stderr)
		os.Exit(1)
	}
	cmdName := args[0]
	cmdArgs := args[1:]
	if configFile != "" {
		config, err = ini.LoadFile(configFile)
	} else {
		for _, filename := range defaultConfigFiles {
			if config, err = ini.LoadFile(filename); err == nil {
				break
			}
		}
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error] %v\n", err.Error())
		os.Exit(1)
	}
	e := env.Environment{
		Debug:   debug,
		Verbose: verbose,
		Config:  &config,
	}
	defer e.Close()
	err = e.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[Error]: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if e.Debug {
			return
		}
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "[Error]: %v\n", err)
		}
	}()
	os.Exit(cmd.Run(&e, cmdName, cmdArgs))
}
