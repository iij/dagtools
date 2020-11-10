package env

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"runtime"
	"time"

	"github.com/iij/dagtools/ini"
)

var (
	// Version of dagtools.
	Version = "1.7.0-dev"
)

// Environment defines parameters for dagtools
type Environment struct {
	Version     string
	Verbose     bool
	Debug       bool
	Concurrency int
	Config      *ini.Config
	Logger      *log.Logger
	startTime   time.Time
}

// Init do initializing Environment
func (e *Environment) Init() (err error) {
	e.Version = Version
	// debug
	if !e.Debug {
		e.Debug = e.Config.GetBool("dagtools", "debug", false)
	}
	// verbose
	if !e.Verbose {
		e.Verbose = e.Config.GetBool("dagtools", "verbose", true)
	}
	// Set startTime
	e.startTime = time.Now()
	var logger *log.Logger
	t := e.Config.Get("logging", "type", "none")
	flag := log.LstdFlags
	if e.Debug {
		flag = flag | log.Lmicroseconds
	}
	switch t {
	case "file":
		filename := e.Config.Get("logging", "file", "dagtools.log")
		file, _ := os.OpenFile(filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
		logger = log.New(file, "", flag)
		log.SetOutput(file)
	case "stdout":
		logger = log.New(os.Stdout, "", flag)
		log.SetOutput(os.Stdout)
	case "stderr":
		logger = log.New(os.Stderr, "", flag)
		log.SetOutput(os.Stderr)
	case "", "none":
		out := ioutil.Discard
		logger = log.New(out, "", 0)
		log.SetOutput(out)
	default:
		err = errors.New("environment::logging type is invalid")
		return
	}
	e.Logger = logger
	e.Concurrency = e.Config.GetInt("dagtools", "concurrency", 1)
	runtime.GOMAXPROCS(	e.Concurrency)

	if e.Debug {
		logger.Println("Environment:", e.String())
		exp, _ := regexp.Compile(`"secretAccessKey":"[^"]+"`)
		logger.Println("Config:", exp.ReplaceAllString(e.Config.String(), `"secretAccessKey":"..."`))
	}
	return nil
}

func (e *Environment) String() string {
	return fmt.Sprintf("{Version: %s, Verbose: %v, Debug: %v, Concurrency: %d}", e.Version, e.Verbose, e.Debug, e.Concurrency)
}

// GetElapsedTimeMs returns elapsed time (milli seconds)
func (e *Environment) GetElapsedTimeMs() int64 {
	return (time.Now().UnixNano() - e.startTime.UnixNano()) / 1000000
}

// Close do cleanup Environment
func (e *Environment) Close() {
	e.Logger.Println("Environment closed.")
}
