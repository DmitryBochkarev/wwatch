package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"regexp"
	"time"
)

type Config struct {
	Dir      string   `toml:"dir"`
	Cwd      string   `toml:"cwd"`
	Cmd      string   `toml:"cmd"`
	CmdArgs  []string `toml:"args"`
	Kill     string   `toml:"kill"`
	KillArgs []string `toml:"kill_args"`
	Match    string   `toml:"match"`
	Delay    string   `toml:"delay"`
	Run      map[string]*Config

	parent *Config
}

func (c *Config) Load(file string) {
	configData, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	if _, err := toml.Decode(string(configData), c); err != nil {
		panic(err)
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

func (c *Config) GetKill() string {
	switch {
	case c.Kill != "":
		return c.Kill
	case c.parent != nil:
		return c.parent.GetKill()
	default:
		return ""
	}
}

func (c *Config) GetKillArgs() []string {
	switch {
	case len(c.KillArgs) > 0:
		return c.KillArgs
	case c.parent != nil:
		return c.parent.GetKillArgs()
	default:
		return c.KillArgs
	}
}

func (c *Config) GetMatch() regexp.Regexp {
	switch {
	case c.Match != "":
		rx, err := regexp.Compile(c.Match)
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
		return DEFAULT_DELAY
	}
}

func (c *Config) Tasks() (*map[string]*Task, error) {
	tasks := make(map[string]*Task)
	switch {
	case c.GetCmd() != nil && len(c.Run) > 0:
		panic(errors.New("You can't specify tasks in main and run sections at the same time."))
	case c.GetCmd() != "":
		task, err := NewTask(c)
		if err != nil {
			panic(err)
		}
		tasks["default"] = task
	case len(c.Run) > 0:
		for name, run := range c.Run {
			run.parent = c
			task, err := NewTask(run)
			if err != nil {
				panic(err)
			}
			tasks[name] = task
		}
	default:
		panic(errors.New("Task not found"))
	}
	return &tasks
}

func (c Config) String() string {
	return fmt.Sprintf("dir: %s, cwd: %s, cmd: %s, cmdArgs: %q, kill: %s, match: %s, delay: %s",
		c.GetDir(), c.GetCwd(), c.GetCmd(), c.GetCmdArgs(), c.GetKill(), c.GetMatch(), c.GetDelay())
}
