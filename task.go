package main

import (
	"github.com/howeyc/fsnotify"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"
	"syscall"
	"time"
)

type Task struct {
	Dir            string
	Cwd            string
	Cmd            string
	CmdArgs        []string
	OnStartCmd     string
	OnStartCmdArgs []string
	PidFile        string
	Match          *regexp.Regexp
	Ignore         *regexp.Regexp
	After          bool
	Delay          time.Duration
	Recursive      bool
	DotFiles       bool

	watchers []*fsnotify.Watcher
	command  *exec.Cmd
	mx       sync.Mutex
}

func NewTask(c *Config) (*Task, error) {
	task := &Task{
		Dir:            c.GetDir(),
		Cwd:            c.GetCwd(),
		Cmd:            c.GetCmd(),
		CmdArgs:        c.GetCmdArgs(),
		OnStartCmd:     c.GetOnStartCmd(),
		OnStartCmdArgs: c.GetOnStartCmdArgs(),
		PidFile:        c.GetPidFile(),
		Match:          c.GetMatch(),
		Ignore:         c.GetIgnore(),
		After:          c.GetAfter(),
		Delay:          c.GetDelay(),
		Recursive:      c.GetRecursive(),
		DotFiles:       c.GetDotFiles(),
	}
	return task, nil
}

func (t *Task) StartWatch(event chan *fsnotify.FileEvent) {
	t.mx.Lock()
	defer t.mx.Unlock()

	if !t.Recursive {
		watcher, err := startWatcher(t.Dir, event)
		if err != nil {
			log.Fatal(err)
		}
		t.watchers = append(t.watchers, watcher)
		return
	}

	filepath.Walk(t.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Print(err)
			return nil
		}

		if !info.IsDir() {
			return nil
		}

		if path != "." && !t.DotFiles && isDotfile(path) {
			return filepath.SkipDir
		}

		if t.Ignore.MatchString(path) {
			return filepath.SkipDir
		}

		watcher, err := startWatcher(path, event)
		if err != nil {
			log.Fatal(err)
		}
		t.watchers = append(t.watchers, watcher)
		return nil
	})
}

func (t *Task) StopWatch() {
	t.mx.Lock()
	defer t.mx.Unlock()

	watchers := t.watchers
	t.watchers = []*fsnotify.Watcher{}

	for _, watcher := range watchers {
		watcher.Close()
	}
}

func (t *Task) Run() {
	if t.OnStartCmd != "" {
		exe := os.Expand(t.OnStartCmd, os.Getenv)
		var args = make([]string, len(t.OnStartCmdArgs))
		for i, arg := range t.OnStartCmdArgs {
			args[i] = os.Expand(arg, os.Getenv)
		}

		log.Printf("run onstart command %s %v\n", exe, args)
		command := exec.Command(exe, args...)
		command.Dir = t.Cwd
		command.Stdout = os.Stdout
		command.Stderr = os.Stderr
		command.Run()
	}

	if !t.After {
		t.Exec()
	}

	event := make(chan *fsnotify.FileEvent)
	t.StartWatch(event)

	var timer <-chan time.Time
	for {
		select {
		case ev := <-event:
			path := ev.Name
			if ev.IsCreate() || ev.IsDelete() || ev.IsRename() {
				t.StopWatch()
				t.StartWatch(event)
			}

			if !t.DotFiles && isDotfile(path) {
				break
			}

			if !t.Match.MatchString(path) || t.Ignore.MatchString(path) {
				break
			}

			log.Printf("File changed(%s)", ev.String())

			if t.Delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before rerun...\n", t.Delay)
			}

			timer = time.After(t.Delay)
		case <-timer:
			t.Stop()
			t.Exec()
		}
	}
}

func (t *Task) Exec() {
	t.mx.Lock()
	defer t.mx.Unlock()
	exe := os.Expand(t.Cmd, os.Getenv)

	var args = make([]string, len(t.CmdArgs))
	for i, arg := range t.CmdArgs {
		args[i] = os.Expand(arg, os.Getenv)
	}

	log.Printf("run %s %v\n", exe, args)
	t.command = exec.Command(exe, args...)
	t.command.Dir = t.Cwd
	t.command.Stdout = os.Stdout
	t.command.Stderr = os.Stderr
	t.command.Start()
	go t.command.Wait()
}

func (t *Task) Stop() {
	t.mx.Lock()
	defer t.mx.Unlock()

	if t.command == nil {
		return
	}

	log.Println("stop")

	if t.command.ProcessState != nil && t.command.ProcessState.Exited() {
		return
	}

	processPid := t.command.Process.Pid
	groupPid := -1 * processPid

	if t.PidFile != "" {
		pidBites, err := ioutil.ReadFile(t.PidFile)
		if err != nil {
			log.Fatalf("error while reading pid file(%s): %q", t.PidFile, err)
		}
		processPid, err = strconv.Atoi(string(pidBites))
		if err != nil {
			log.Fatalf("error while parsing pid file(%s): %q", t.PidFile, err)
		}

	}

	if t.command.ProcessState != nil {
		return
	}

	stopProcess(groupPid, processPid, (t.PidFile != ""), time.Duration(1*time.Second))
	t.command.Wait()
}

func stopProcess(groupPid, processPid int, useProcessPid bool, wait time.Duration) {
	log.Printf("send SIGTERM to process group %d\n", groupPid)
	group, err := os.FindProcess(groupPid)
	if err != nil {
		log.Fatal(err)
	}

	group.Signal(syscall.SIGTERM)

	go func() {
		time.Sleep(wait)
		group.Kill()
	}()

	if useProcessPid {
		proc, err := os.FindProcess(processPid)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("send SIGINT to process pid %d\n", processPid)
		proc.Signal(os.Interrupt)
		go func() {
			time.Sleep(wait)
			proc.Kill()
		}()
		proc.Wait()
	}

	group.Wait()
}

func (t *Task) Shutdown() {
	t.StopWatch()
	t.Stop()
}
