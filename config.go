package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Config struct {
	Dir       string   `toml:"dir"`
	Cwd       string   `toml:"cwd"`
	Cmd       string   `toml:"cmd"`
	CmdArgs   []string `toml:"args"`
	PidFile   string   `toml:"pidfile"`
	Match     string   `toml:"match"`
	Ext       string   `toml:"ext"`
	Delay     string   `toml:"delay"`
	Recursive *bool    `toml:"recursive"`
	DotFiles  *bool    `toml:"dotfiles"`

	Run map[string]Config

	configFile string
	parent     *Config
}

func (c *Config) Load(file string) {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	c.configFile = resolvePath(cwd, file)
	configData, err := ioutil.ReadFile(c.configFile)
	if err != nil {
		log.Fatal(err)
	}
	if _, err := toml.Decode(string(configData), c); err != nil {
		log.Fatal(err)
	}
}

func (c *Config) GetConfigPath() string {
	switch {
	case c.configFile != "":
		return filepath.Dir(c.configFile)
	case c.parent != nil:
		return c.parent.GetConfigPath()
	default:
		return ""
	}
}

func (c *Config) ResolveFilepath(relativePath string) string {
	path := relativePath
	if configPath := c.GetConfigPath(); configPath != "" {
		path = resolvePath(configPath, path)
	}

	return path
}

func (c *Config) GetDir() string {
	var dir string
	switch {
	case c.Dir != "":
		dir = c.Dir
	case c.parent != nil:
		dir = c.parent.GetDir()
	default:
		dir = DEFAULT_DIR
	}

	dir = c.ResolveFilepath(dir)

	return dir
}

func (c *Config) GetCwd() string {
	var dir string
	switch {
	case c.Cwd != "":
		dir = c.Cwd
	case c.parent != nil:
		dir = c.parent.GetCwd()
	default:
		dir = DEFAULT_CWD
	}

	dir = c.ResolveFilepath(dir)

	return dir
}

func (c *Config) GetCmd() string {
	return c.Cmd
}

func (c *Config) GetCmdArgs() []string {
	return c.CmdArgs
}

func (c *Config) GetPidFile() string {
	if c.PidFile != "" {
		return c.ResolveFilepath(c.PidFile)
	}
	return ""
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
		tasks["default"] = task
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
