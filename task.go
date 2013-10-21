package main

import (
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

type Task struct {
	Dir      string
	Cwd      string
	Cmd      string
	CmdArgs  []string
	Kill     string
	KillArgs []string
	Match    *regexp.Regexp
	Delay    time.Duration
	Recursive    bool

	command *exec.Cmd
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
		Recursive: c.GetRecursive(),
	}
	return task, nil
}

func (t *Task) Run(done chan bool) {
	t.Exec()

	var timer <-chan time.Time

	quit := make(chan bool)
	event := make(chan *fsnotify.FileEvent)

	startWatch(t, quit, event)

	var rerunMx sync.Mutex
	for {
		select {
		case ev := <-event:
			if ev.IsCreate() || ev.IsDelete() || ev.IsRename() {
				close(quit)
				quit = make(chan bool)
				startWatch(t, quit, event)
			}

			if !t.Match.MatchString(ev.Name) {
				break
			}

			log.Printf("File changed(%s)", ev.String())
			if t.Delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before rerun...\n", t.Delay)
			}

			timer = time.After(t.Delay)
		case <-timer:
			rerunMx.Lock()
			t.Stop()
			t.Exec()
			rerunMx.Unlock()
		case <-done:
			t.Stop()
			return
		}
	}
}

func (t *Task) Exec() {
	exe := os.Expand(t.Cmd, os.Getenv)

	var args = make([]string, len(t.CmdArgs))
	for i, arg := range t.CmdArgs {
		args[i] = os.Expand(arg, os.Getenv)
	}

	log.Println(t.Cwd)
	log.Printf("run %s %v\n", exe, args)
	t.command = exec.Command(exe, args...)
	t.command.Dir = t.Cwd
	t.command.Stdout = os.Stdout
	t.command.Stderr = os.Stderr
	t.command.Start()
	go t.command.Wait()
}

func (t *Task) Stop() {
	if t.command == nil {
		log.Fatal("Trying to stop not runned process")
	}

	if t.command.ProcessState != nil && t.command.ProcessState.Exited() {
		return
	}

	if t.Kill == "" {
		if t.command.ProcessState == nil {
			t.command.Process.Signal(os.Interrupt)
		}
		t.command.Wait()
		return
	}

	exe := os.Expand(t.Kill, os.Getenv)

	var args = make([]string, len(t.KillArgs))
	for i, arg := range t.KillArgs {
		args[i] = os.Expand(arg, func(key string) string {
			if key == "WWATCH_PID" {
				return fmt.Sprintf("%d", t.command.Process.Pid)
			}
			return os.Getenv(key)
		})
	}

	log.Printf("run %s %v\n", exe, args)
	cmdKill := exec.Command(exe, args...)
	cmdKill.Dir = t.Cwd
	cmdKill.Stdout = os.Stdout
	cmdKill.Stderr = os.Stderr
	cmdKill.Run()

	t.command.Wait()
}
