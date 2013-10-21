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
	"sync"
	"time"
)

const (
	VERSION               = "0.6.5"
	DEFAULT_DIR           = "."
	DEFAULT_CWD           = "."
	DEFAULT_MATCH_PATTERN = ".*"
	DEFAULT_DELAY         = time.Duration(100 * time.Millisecond)
)

var (
	commandLineDir, commandLineCwd, commandLineMatchPattern string
	commandLineDelay                                        time.Duration
	commandLineCommand, commandLineKill                     string
	commandLineConfig                                       string
	commandLinePrintVersion                                 bool

	config Config
	tasks  *map[string]*Task
)

func init() {
	flag.StringVar(&commandLineDir, "dir", DEFAULT_DIR, "directory to watch")
	flag.StringVar(&commandLineCwd, "cwd", DEFAULT_CWD, "current working directory")
	flag.StringVar(&commandLineMatchPattern, "match", DEFAULT_MATCH_PATTERN, "file(fullpath) match regexp")
	flag.DurationVar(&commandLineDelay, "delay", DEFAULT_DELAY, "delay before rerun cmd")
	flag.StringVar(&commandLineCommand, "cmd", "", "command to run")
	flag.StringVar(&commandLineKill, "kill", "", "command to shutdown process. Example: kill -9 $WWATCH_PID. Default send INT signal.")
	flag.StringVar(&commandLineConfig, "config", "", "path to configuration file(*.toml)")
	flag.BoolVar(&commandLinePrintVersion, "version", false, "print version")
}

func main() {
	flag.Parse()

	log.SetPrefix("wwatch ")

	if printVertion {
		log.Fatalf("version: %s", VERSION)
	}

	switch {
	case commandString == "" && configFile == "":
		log.Fatal("You should specify command or path to configuration file")
	case commandString != "":
		config.Dir = dir
		config.Cwd = cwd
		config.Match = matchPattern
		config.Delay = delay
		cmd, cmdArgs := parseCommandString(commandString)
		config.Cmd = cmd
		config.CmdArgs = cmdArgs
		if shutdownString != "" {
			kill, killArgs := parseCommandString(commandString)
			config.Kill = kill
			config.KillArgs = killArgs
		}
	case configFile != "":
		config.Load(configFile)
	}

	tasks, err := config.Tasks()

	if err != nil {
		log.Fatal(err)
	}

	matchRx, err = regexp.Compile(matchPattern)

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

	var rerunMx sync.Mutex

	for {
		select {
		case ev := <-event:
			if ev.IsCreate() || ev.IsDelete() || ev.IsRename() {
				close(quit)
				quit = make(chan bool)
				startWatch(dir, quit, event)
			}

			if !matchRx.MatchString(ev.Name) {
				break
			}

			log.Printf("File changed(%s)", ev.String())

			if delay >= time.Duration(500*time.Millisecond) {
				log.Printf("wait %s before rerun...\n", delay)
			}

			timer = time.After(delay)
		case <-timer:
			rerunMx.Lock()
			stopCommand(cmd, shutdownString)
			cmd = execCommand(commandString, cwd)
			rerunMx.Unlock()
		case signal := <-done:
			fmt.Println("Got signal:", signal)
			stopCommand(cmd, shutdownString)
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
			cmd.Process.Signal(os.Interrupt)
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
