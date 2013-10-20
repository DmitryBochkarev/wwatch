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

var (
	dir, commandString, matchPattern string
	delay                            time.Duration
)

func init() {
	flag.StringVar(&dir, "dir", ".", "directory to watch")
	flag.StringVar(&commandString, "cmd", "", "command to run")
	flag.StringVar(&matchPattern, "match", ".*", "file(fullpath) match regexp")
	flag.DurationVar(&delay, "delay", time.Duration(100*time.Millisecond), "delay before rerun cmd")
}

func main() {
	flag.Parse()

	log.SetPrefix("watch")
	if commandString == "" {
		log.Fatal("You should specify command.")
	}
	matchRx, err := regexp.Compile(matchPattern)
	if err != nil {
		log.Fatal(err)
	}
	log.SetPrefix(fmt.Sprintf("watch %s ", matchPattern))
	cmd := execCommand(commandString)

	done := make(chan bool)
	quit := make(chan bool)
	event := make(chan *fsnotify.FileEvent)

	startWatch(dir, quit, event)

	for {
		select {
		case ev := <-event:
			close(quit)
			quit = make(chan bool)

			if !matchRx.MatchString(ev.Name) {
				startWatch(dir, quit, event)
				continue
			}

			log.Printf("File changed: %s\n", ev.Name)
			stopCommand(cmd)
			if delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before run...\n", delay)
			}
			time.Sleep(delay)
			cmd = execCommand(commandString)
			startWatch(dir, quit, event)
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
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
		return
	}

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
			log.Println("watch error:", err)
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

func execCommand(commandString string) *exec.Cmd {
	exe, args := parseCommandString(commandString)
	log.Printf("run %s %v\n", exe, args)
	cmd := exec.Command(exe, args...)
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
