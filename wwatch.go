package main

import (
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	VERSION = "0.6.0"
)

var (
	dir, commandString, matchPattern string
	cwd                              string
	delay                            time.Duration
	printVertion                     bool
)

func init() {
	flag.StringVar(&dir, "dir", ".", "directory to watch")
	flag.StringVar(&commandString, "cmd", "", "command to run")
	flag.StringVar(&matchPattern, "match", ".*", "file(fullpath) match regexp")
	flag.StringVar(&cwd, "cwd", ".", "current working directory")
	flag.DurationVar(&delay, "delay", time.Duration(100*time.Millisecond), "delay before rerun cmd")
	flag.BoolVar(&printVertion, "version", false, "print version")
}

func main() {
	flag.Parse()

	log.SetPrefix("wwatch ")

	if printVertion {
		log.Fatalf("version: %s", VERSION)
	}
	if commandString == "" {
		log.Fatal("You should specify command(-cmd='cal')")
	}

	matchRx, err := regexp.Compile(matchPattern)

	if err != nil {
		log.Fatal(err)
	}

	log.SetPrefix(fmt.Sprintf("wwatch %s ", matchPattern))

	cmd := execCommand(commandString, cwd)

	var timer <-chan time.Time

	done := make(chan bool)
	quit := make(chan bool)
	event := make(chan *fsnotify.FileEvent)

	startWatch(dir, quit, event)

	for {
		select {
		case ev := <-event:
			removeWatchers := false

			if ev.IsCreate() || ev.IsDelete() || ev.IsRename() {
				removeWatchers = true
			}

			if removeWatchers {
				close(quit)
				quit = make(chan bool)
			}

			if removeWatchers {
				startWatch(dir, quit, event)
			}

			if !matchRx.MatchString(ev.Name) {
				break
			}

			log.Printf("File changed(%s)", ev.String())

			stopCommand(cmd)

			if delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before run...\n", delay)
			}

			timer = time.After(delay)
		case <-timer:
			cmd = execCommand(commandString, cwd)
		case <-done:
			close(quit)
			return
		}
	}
}

func startWatch(dir string, quit chan bool, event chan *fsnotify.FileEvent) {
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Print(err)
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		go startWatcher(path, quit, event)
		return nil
	})
}

func startWatcher(dir string, quit chan bool, event chan *fsnotify.FileEvent) {
	// log.Printf("Define watcher %s", dir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
		return
	}

	// defer log.Printf("Remove watcher %s", dir)
	defer watcher.Close()

	err = watcher.Watch(dir)

	if err != nil {
		log.Print(err)
		return
	}

	for {
		select {
		case ev := <-watcher.Event:
			event <- ev
		case err := <-watcher.Error:
			log.Println("watch error: ", err)
		case <-quit:
			return
		}
	}
}

func parseCommandString(commandString string) (exe string, args []string) {
	cmdSplit := strings.SplitN(commandString, " ", 2)
	exe = cmdSplit[0]
	if len(cmdSplit) > 1 {
		args = strings.Split(cmdSplit[1], " ")
	}
	return
}

func execCommand(commandString, cwd string) *exec.Cmd {
	exe, args := parseCommandString(commandString)
	log.Printf("run %s %v\n", exe, args)
	cmd := exec.Command(exe, args...)
	cmd.Dir = cwd
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Start()
	go cmd.Wait()
	return cmd
}

func stopCommand(cmd *exec.Cmd) {
	if cmd.ProcessState == nil {
		cmd.Process.Kill()
	}
	cmd.Wait()
}
