package main

import (
	"flag"
	"fmt"
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	VERSION = "0.6.2"
)

var (
	dir, commandString, matchPattern string
	cwd, shutdownString              string
	delay                            time.Duration
	printVertion                     bool
)

func init() {
	flag.StringVar(&dir, "dir", ".", "directory to watch")
	flag.StringVar(&commandString, "cmd", "", "command to run")
	flag.StringVar(&shutdownString, "kill", "", "command to shutdown process. Example: kill -9 $WWATCH_PID")
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

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, os.Kill)

	cmd := execCommand(commandString, cwd)

	var timer <-chan time.Time

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

			stopCommand(cmd, shutdownString)

			if delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before run...\n", delay)
			}

			timer = time.After(delay)
		case <-timer:
			cmd = execCommand(commandString, cwd)
		case signal := <-done:
			fmt.Println("Got signal:", signal)
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
	commandString = os.Expand(commandString, os.Getenv)

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

func stopCommand(cmd *exec.Cmd, shutdownString string) {
	if shutdownString == "" {
		if cmd.ProcessState == nil {
			cmd.Process.Kill()
		}
		cmd.Wait()
		return
	}

	shutdownString = os.Expand(shutdownString, func(v string) string {
		if v == "WWATCH_PID" {
			return fmt.Sprintf("%d", cmd.Process.Pid)
		}
		return fmt.Sprintf("${%s}", v)
	})

	shutdownString = os.Expand(shutdownString, os.Getenv)

	exe, args := parseCommandString(shutdownString)
	log.Printf("run %s %v\n", exe, args)
	cmdKill := exec.Command(exe, args...)
	cmdKill.Dir = cwd
	cmdKill.Stdout = os.Stdout
	cmdKill.Stderr = os.Stderr
	cmdKill.Run()

	cmd.Wait()
}
