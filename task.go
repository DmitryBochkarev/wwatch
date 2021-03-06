package main

import (
	"github.com/howeyc/fsnotify"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type Task struct {
	Dir        string
	Cwd        string
	Cmd        []string
	OnStartCmd []string
	PidFile    string
	Match      *regexp.Regexp
	Ignore     *regexp.Regexp
	After      bool
	Delay      time.Duration
	Recursive  bool
	DotFiles   bool

	Stdout io.Writer
	Stderr io.Writer

	name     string
	watchers []*fsnotify.Watcher
	command  *exec.Cmd
	mx       sync.Mutex
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
	if len(t.OnStartCmd) > 0 {
		var args = make([]string, len(t.OnStartCmd))
		for i, arg := range t.OnStartCmd {
			args[i] = os.Expand(arg, os.Getenv)
		}

		log.Printf("%s run onstart command %s", t.name, strings.Join(args, " "))
		command := exec.Command(args[0], args[1:]...)
		command.Dir = t.Cwd
		command.Stdout = t.Stdout
		command.Stderr = t.Stderr
		command.Run()
	}

	if !t.After {
		t.Exec()
	}

	event := make(chan *fsnotify.FileEvent)
	t.StartWatch(event)

	prevPath := ""

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

			if prevPath != path {
				log.Printf("%s", path)

				if t.Delay >= time.Duration(500*time.Millisecond) {
					log.Printf("%s wait %s before rerun...\n", t.name, t.Delay)
				}

				prevPath = path
			}

			timer = time.After(t.Delay)
		case <-timer:
			prevPath = ""
			t.Stop()
			t.Exec()
		}
	}
}

func (t *Task) Exec() {
	t.mx.Lock()
	defer t.mx.Unlock()

	var args = make([]string, len(t.Cmd))
	for i, arg := range t.Cmd {
		args[i] = os.Expand(arg, os.Getenv)
	}

	log.Printf("%s run: %s", t.name, strings.Join(args, " "))
	t.command = exec.Command(args[0], args[1:]...)
	t.command.Dir = t.Cwd
	t.command.Stdout = t.Stdout
	t.command.Stderr = t.Stderr
	t.command.Start()
	go func() {
		t.command.Wait()
		log.Printf("%s process exited", t.name)
	}()
}

func (t *Task) Stop() {
	t.mx.Lock()
	defer t.mx.Unlock()

	if t.command == nil {
		return
	}

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
	log.Printf("send SIGTERM to process group %d", groupPid)
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

		log.Printf("send SIGINT to process pid %d", processPid)
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
