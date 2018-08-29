package ini

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	commentRegexp = regexp.MustCompile(`[;#][^"]*$`)
	sectionRegexp = regexp.MustCompile(`^\[(.*)]`)
	optionRegexp  = regexp.MustCompile(`^([^=:]+)[=:](.*)$`)
)

// InvalidSyntax represents syntax error when parsing INI file.
type InvalidSyntax struct {
	Config *Config
	Line   int
}

func (e InvalidSyntax) Error() string {
	return fmt.Sprintf("syntax error in %s (line: %d)", e.Config.Filename, e.Line)
}

// Config contains all section in INI file.
type Config struct {
	Filename string
	Sections map[string]Section
}

// String returns the config values as JSON string
func (c *Config) String() string {
	dat, _ := json.Marshal(c.Sections)
	return fmt.Sprintf("{file: %q, content: %s}", c.Filename, string(dat))
}

// Section returns Section of specified name
func (c *Config) Section(name string) *Section {
	section := c.Sections[name]
	if section == nil {
		section = Section{}
	}
	return &section
}

// HasSection returns true if contains the section in config
func (c *Config) HasSection(name string) bool {
	return c.Sections[name] != nil
}

// NewSection returns an initiated blank Section
func (c *Config) NewSection(name string) *Section {
	section := c.Sections[name]
	if section == nil {
		section = *c.Section(name)
		c.Sections[name] = section
	}
	return &section
}

// Get returns a option value as string
func (c *Config) Get(section string, key string, defaultValue string) string {
	return c.Section(section).Get(key, defaultValue)
}

// GetBool returns a option value as bool
func (c *Config) GetBool(section string, key string, defaultValue bool) bool {
	return c.Section(section).GetBool(key, defaultValue)
}

// GetInt returns a option value as int64
func (c *Config) GetInt(section string, key string, defaultValue int) int {
	return c.Section(section).GetInt(key, defaultValue)
}

// GetInt64 returns a option value as int64
func (c *Config) GetInt64(section string, key string, defaultValue int64) int64 {
	return c.Section(section).GetInt64(key, defaultValue)
}

// Set config
func (c *Config) Set(section string, key string, value string) {
	if !c.HasSection(section) {
		c.NewSection(section)
	}
	c.Section(section).Set(key, value)
}

// Section is a group of options.
type Section map[string]string

// Get returns a option value as string
func (s *Section) Get(key string, defaultValue string) string {
	if value, ok := (*s)[key]; ok {

		return value
	}
	return defaultValue
}

// GetBool returns a option value as bool
func (s *Section) GetBool(key string, defaultValue bool) bool {
	_default := "false"
	if defaultValue {
		_default = "true"
	}
	v := s.Get(key, _default)
	return v == "true" || v == "1"
}

// GetInt returns a option value as int64
func (s *Section) GetInt(key string, defaultValue int) int {
	i, _ := strconv.ParseInt(s.Get(key, strconv.FormatInt(int64(defaultValue), 10)), 10, 0)
	return int(i)
}

// GetInt64 returns a option value as int64
func (s *Section) GetInt64(key string, defaultValue int64) int64 {
	i, _ := strconv.ParseInt(s.Get(key, strconv.FormatInt(defaultValue, 10)), 10, 0)
	return i
}

// Set config
func (s *Section) Set(key string, value string) {
	(*s)[key] = value
}

func parseFile(c Config) error {
	in, err := os.Open(c.Filename)
	if err != nil {
		return err
	}
	bufin := bufio.NewReader(in)
	defer in.Close()
	var (
		section   *Section
		line      string
		wasPrefix = false
		lineNum   = 0
	)
	for {
		buf, isPrefix, err := bufin.ReadLine()
		if err != nil {
			break
		}
		if wasPrefix {
			line += string(buf)
		} else {
			line = string(buf)
		}
		if !isPrefix {
			line = strings.TrimSpace(line)
			line = commentRegexp.ReplaceAllString(line, "")
			lineNum++
			if len(line) == 0 {
				// Skip blank lines
				continue
			}
			if groups := sectionRegexp.FindStringSubmatch(line); groups != nil {
				name := strings.TrimSpace(groups[1])
				section = c.NewSection(name)
			} else if groups := optionRegexp.FindStringSubmatch(line); groups != nil {
				key, val := strings.TrimSpace(groups[1]), strings.TrimSpace(groups[2])
				if section != nil {
					section.Set(key, strings.Trim(val, `"`))
				}
			} else {
				return InvalidSyntax{Config: &c, Line: lineNum}
			}
		}
		wasPrefix = isPrefix
	}
	return nil
}

// LoadFile returns Config of loaded INI file.
func LoadFile(filename string) (Config, error) {
	c := Config{Filename: filename, Sections: make(map[string]Section)}
	err := parseFile(c)
	return c, err
}
