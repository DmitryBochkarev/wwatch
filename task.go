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
	Dir       string
	Cwd       string
	Cmd       string
	CmdArgs   []string
	PidFile   string
	Match     *regexp.Regexp
	Delay     time.Duration
	Recursive bool
	DotFiles  bool

	watchersCh chan bool
	command    *exec.Cmd
}

func NewTask(c *Config) (*Task, error) {
	task := &Task{
		Dir:       c.GetDir(),
		Cwd:       c.GetCwd(),
		Cmd:       c.GetCmd(),
		CmdArgs:   c.GetCmdArgs(),
		PidFile:   c.GetPidFile(),
		Match:     c.GetMatch(),
		Delay:     c.GetDelay(),
		Recursive: c.GetRecursive(),
		DotFiles:  c.GetDotFiles(),
	}
	return task, nil
}

func (t *Task) StartWatch(event chan *fsnotify.FileEvent) {
	t.watchersCh = make(chan bool)

	if !t.Recursive {
		startWatcher(t.Dir, t.watchersCh, event)
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

		startWatcher(path, t.watchersCh, event)
		return nil
	})
}

func (t *Task) StopWatch() {
	close(t.watchersCh)
}

func (t *Task) Run() {
	t.Exec()

	var timer <-chan time.Time

	event := make(chan *fsnotify.FileEvent)

	t.StartWatch(event)

	var rerunMx sync.Mutex
	for {
		select {
		case ev := <-event:
			if ev.IsCreate() || ev.IsDelete() || ev.IsRename() {
				t.StopWatch()
				t.StartWatch(event)
			}

			if !t.DotFiles && isDotfile(ev.Name) {
				break
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
		}
	}
}

func (t *Task) Exec() {
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
	if t.command == nil {
		log.Fatal("Trying to stop not runned process")
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

	log.Printf("send SIGTERM to process group %d\n", groupPid)
	group, err := os.FindProcess(groupPid)
	if err != nil {
		log.Fatal(err)
	}

	group.Signal(syscall.SIGTERM)
	group.Wait()

	if t.PidFile != "" {
		proc, err := os.FindProcess(processPid)
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("send SIGINT to process pid %d(%s)\n", processPid, t.PidFile)
		proc.Signal(os.Interrupt)
		proc.Wait()
	}

	t.command.Wait()
}

func (t *Task) Shutdown() {
	t.StopWatch()
	t.Stop()
}
