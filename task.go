package main

import (
	"fmt"
	"regexp"
	"time"
)

type Task struct {
	Dir      string
	Cwd      string
	Cmd      string
	CmdArgs  []string
	Kill     string
	KillArgs []string
	Match    regexp.Regexp
	Delay    time.Duration
}

func NewTask(c *Config) (*Task, error) {
	task := &Task{
		Dir:      c.GetDir(),
		Cwd:      c.GetCwd(),
		Cmd:      c.GetCmd(),
		CmdArgs:  c.GetCmdArgs(),
		Kill:     c.GetKill(),
		KillArgs: c.GetKillArgs(),
		Match:    c.GetMatch(),
		Delay:    c.GetDelay(),
	}
	return task, nil
}
