package cmd

import (
	"errors"
	"flag"
	"fmt"
	"math"
	"time"
)

var (
	units = []string{"B", "KB", "MB", "GB", "TB", "PB"}
	// ErrArgument represents error in command argument
	ErrArgument = errors.New("argument error")
)

// HumanReadableBytes returns a size string as human-readable format (e.x., 1024 -> 1KB)
func HumanReadableBytes(s uint64) string {
	var base float64 = 1024
	if s < 10 {
		return fmt.Sprintf("%dB", s)
	}
	e := math.Floor(math.Log(float64(s)) / math.Log(base))
	suffix := units[int(e)]
	val := math.Floor(float64(s)/math.Pow(base, e)*10+0.5) / 10
	f := "%.0f%s"
	if val < 10 {
		f = "%.1f%s"
	}
	return fmt.Sprintf(f, val, suffix)
}

// LocalTimeString returns a formatted date string (2006-01-02 15:04:05)
func LocalTimeString(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05")
}

// OptionUsage returns usage of command line option.
func OptionUsage(f *flag.FlagSet) string {
	var usage = ""
	f.VisitAll(func(flag *flag.Flag) {
		format := "  -%s=%s: %s\n"
		usage += fmt.Sprintf(format, flag.Name, flag.DefValue, flag.Usage)
	})
	return usage
}
