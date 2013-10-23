package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Dir            string   `toml:"dir"`
	Cwd            string   `toml:"cwd"`
	Cmd            string   `toml:"cmd"`
	CmdArgs        []string `toml:"args"`
	OnStartCmd     string   `toml:"onstart"`
	OnStartCmdArgs []string `toml:"onstart_args"`
	PidFile        string   `toml:"pidfile"`
	Match          string   `toml:"match"`
	Ext            string   `toml:"ext"`
	Ignore         string   `toml:"ignore"`
	After          *bool    `toml:"after"`
	Delay          string   `toml:"delay"`
	Recursive      *bool    `toml:"recursive"`
	DotFiles       *bool    `toml:"dotfiles"`

	Run map[string]Config

	parent *Config
}

func (c *Config) Load(file string) {
	configData, err := ioutil.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := toml.Decode(string(configData), c); err != nil {
		log.Fatal(err)
	}
}

func (c *Config) GetDir() string {
	switch {
	case c.Dir != "":
		return c.Dir
	case c.parent != nil:
		return c.parent.GetDir()
	default:
		return DEFAULT_DIR
	}
}

func (c *Config) GetCwd() string {
	switch {
	case c.Cwd != "":
		return c.Cwd
	case c.parent != nil:
		return c.parent.GetCwd()
	default:
		return DEFAULT_CWD
	}
}

func (c *Config) GetCmd() string {
	return c.Cmd
}

func (c *Config) GetCmdArgs() []string {
	return c.CmdArgs
}

func (c *Config) GetOnStartCmd() string {
	return c.OnStartCmd
}

func (c *Config) GetOnStartCmdArgs() []string {
	return c.OnStartCmdArgs
}

func (c *Config) GetPidFile() string {
	return c.PidFile
}

func (c *Config) GetMatch() *regexp.Regexp {
	match := c.Match

	if ext := c.GetExt(); ext != "" {
		ext = strings.Replace(ext, " ", "", -1)
		ext = strings.Replace(ext, ",", "|", -1)
		match = fmt.Sprintf(".*\\.(%s)$", ext)
	}

	switch {
	case match != "":
		rx, err := regexp.Compile(match)
		if err != nil {
			panic(err)
		}
		return rx
	case c.parent != nil:
		return c.parent.GetMatch()
	default:
		return regexp.MustCompile(DEFAULT_MATCH_PATTERN)
	}
}

func (c *Config) GetExt() string {
	switch {
	case c.Ext != "":
		return c.Ext
	case c.parent != nil:
		return c.parent.GetExt()
	default:
		return ""
	}
}

func (c *Config) GetIgnore() *regexp.Regexp {
	switch {
	case c.Ignore != "":
		rx, err := regexp.Compile(c.Ignore)
		if err != nil {
			panic(err)
		}
		return rx
	case c.parent != nil:
		return c.parent.GetIgnore()
	default:
		return regexp.MustCompile("^$")
	}
}

func (c *Config) GetAfter() bool {
	switch {
	case c.After != nil:
		return *c.After
	case c.parent != nil:
		return c.parent.GetAfter()
	default:
		return DEFAULT_AFTER_CHANGE
	}
}

func (c *Config) GetDelay() time.Duration {
	switch {
	case c.Delay != "":
		delay, err := time.ParseDuration(c.Delay)
		if err != nil {
			panic(err)
		}
		return delay
	case c.parent != nil:
		return c.parent.GetDelay()
	default:
		delay, err := time.ParseDuration(DEFAULT_DELAY)
		if err != nil {
			panic(err)
		}
		return delay
	}
}

func (c *Config) GetRecursive() bool {
	switch {
	case c.Recursive != nil:
		return *c.Recursive
	case c.parent != nil:
		return c.parent.GetRecursive()
	default:
		return DEFAULT_RECURSIVE
	}
}

func (c *Config) GetDotFiles() bool {
	switch {
	case c.DotFiles != nil:
		return *c.DotFiles
	case c.parent != nil:
		return c.parent.GetDotFiles()
	default:
		return DEFAULT_DOTFILES
	}
}

func (c *Config) Tasks() (*map[string]*Task, error) {
	tasks := make(map[string]*Task)
	switch {
	case c.GetCmd() != "" && len(c.Run) > 0:
		panic(errors.New("You can't specify tasks in main and run sections at the same time."))
	case c.GetCmd() != "":
		task, err := NewTask(c)
		if err != nil {
			panic(err)
		}
		tasks[".default"] = task
	case len(c.Run) > 0:
		for name, run := range c.Run {
			run.parent = c
			task, err := NewTask(&run)
			if err != nil {
				panic(err)
			}
			tasks[name] = task
		}
	default:
		return nil, errors.New("Task not found")
	}
	return &tasks, nil
}
