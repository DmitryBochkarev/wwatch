package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

const (
	DEFAULT_DIR           = "."
	DEFAULT_CWD           = "."
	DEFAULT_MATCH_PATTERN = ".*"
	DEFAULT_DELAY         = "100ms"
	DEFAULT_RECURSIVE     = false
)

var (
	commandLineDir, commandLineCwd, commandLineMatchPattern string
	commandLineDelay                                        string
	commandLineCommand, commandLineKill                     string
	commandLineConfig                                       string
	commandLineRecursive, commandLinePrintVersion           bool

	config Config
	tasks  *map[string]*Task
)

func init() {
	flag.StringVar(&commandLineDir, "dir", DEFAULT_DIR, "directory to watch")
	flag.StringVar(&commandLineCwd, "cwd", DEFAULT_CWD, "current working directory")
	flag.StringVar(&commandLineMatchPattern, "match", DEFAULT_MATCH_PATTERN, "file(fullpath) match regexp")
	flag.StringVar(&commandLineDelay, "delay", DEFAULT_DELAY, "delay before rerun cmd")
	flag.StringVar(&commandLineCommand, "cmd", "", "command to run")
	flag.StringVar(&commandLineKill, "kill", "", "command to shutdown process. Example: kill -9 $WWATCH_PID. Default send INT signal.")
	flag.StringVar(&commandLineConfig, "config", "", "path to configuration file(*.toml)")
	flag.BoolVar(&commandLineRecursive, "recursive", DEFAULT_RECURSIVE, "walk recursive over directories")
	flag.BoolVar(&commandLinePrintVersion, "version", false, "print version")
}

func main() {
	flag.Parse()

	log.SetPrefix("wwatch ")

	if commandLinePrintVersion {
		log.Fatalf("version: %s", VERSION)
	}

	switch {
	case commandLineCommand == "" && commandLineConfig == "":
		log.Fatal("You should specify command or path to configuration file")
	case commandLineCommand != "":
		config.Dir = commandLineDir
		config.Cwd = commandLineCwd
		config.Match = commandLineMatchPattern
		config.Delay = commandLineDelay
		config.Recursive = &commandLineRecursive
		cmd, cmdArgs := parseCommandString(commandLineCommand)
		config.Cmd = cmd
		config.CmdArgs = cmdArgs
		if commandLineKill != "" {
			kill, killArgs := parseCommandString(commandLineKill)
			config.Kill = kill
			config.KillArgs = killArgs
		}
	case commandLineConfig != "":
		config.Load(commandLineConfig)
	}

	tasks, err := config.Tasks()

	if err != nil {
		log.Fatal(err)
	}

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, os.Kill)

	quit := make(chan bool)

	for _, task := range *tasks {
		go task.Run(quit)
	}

	<-done
	close(quit)
}
