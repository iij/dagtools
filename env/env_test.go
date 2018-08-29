package env

import (
	"testing"

	"github.com/iij/dagtools/ini"
)

func TestDefaultEnvironment(t *testing.T) {
	e := Environment{
		Config: &ini.Config{},
	}
	err := e.Init()
	if err != nil {
		t.Error("Environment::type is none in default.")
	}
	if e.Debug != false {
		t.Error("Environment::Debug is false in default.")
	}
	if e.Verbose != true {
		t.Error("Environment::Verbose is true in default.")
	}
	if e.startTime.Unix() == 0 {
		t.Error("Environment::startTime did not set.")
	}
}

func TestInitEnvironment(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	config.Set("dagtools", "verbose", "false")
	config.Set("dagtools", "debug", "true")
	config.Set("logging", "type", "file")
	e := Environment{
		Config: config,
	}
	err := e.Init()
	if err != nil {
		t.Error("Environment::type set file in config.")
	}
	if e.Debug != true {
		t.Error("Environment::Debug set true in config.")
	}
	if e.Verbose != false {
		t.Error("Environment::Verbose set false in config.")
	}
	if e.startTime.Unix() == 0 {
		t.Error("Environment::startTime did not set.")
	}
}

func TestInvalidEnvironment(t *testing.T) {
	config := &ini.Config{Filename: "dummy.ini", Sections: make(map[string]ini.Section)}
	config.Set("logging", "type", "dummy")
	e := Environment{
		Config: config,
	}
	err := e.Init()
	if err.Error() != "environment::logging type is invalid" {
		t.Error("Environment::Debug set true in config.")
	}
}
