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
	DEFAULT_DOTFILES      = false
)

var (
	commandLineDir, commandLineCwd, commandLineMatchPattern, commandLineExt, commandLineIgnorePattern string
	commandLineDelay                                                        string
	commandLineCommand, commandLinePidFile                                  string
	commandLineConfig                                                       string
	commandLineRecursive, commandLineDotFiles, commandLinePrintVersion      bool

	config Config
	tasks  *map[string]*Task
)

func init() {
	flag.StringVar(&commandLineDir, "dir", DEFAULT_DIR, "directory to watch")
	flag.StringVar(&commandLineCwd, "cwd", DEFAULT_CWD, "current working directory")
	flag.StringVar(&commandLineMatchPattern, "match", DEFAULT_MATCH_PATTERN, "file(fullpath) match regexp")
	flag.StringVar(&commandLineExt, "ext", "", "extentions of files to watch: -ext='less,js,coffee'")
	flag.StringVar(&commandLineIgnorePattern, "ignore", "", "regexp patter for ignore watch")
	flag.StringVar(&commandLineDelay, "delay", DEFAULT_DELAY, "delay before rerun cmd")
	flag.StringVar(&commandLineCommand, "cmd", "", "command to run")
	flag.StringVar(&commandLinePidFile, "pidfile", "", "file that content pid of running process")
	flag.StringVar(&commandLineConfig, "config", "", "path to configuration file(*.toml)")
	flag.BoolVar(&commandLineRecursive, "recursive", DEFAULT_RECURSIVE, "walk recursive over directories")
	flag.BoolVar(&commandLineDotFiles, "dotfiles", DEFAULT_DOTFILES, "watch on dotfiles")
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
		config.Ext = commandLineExt
		config.Delay = commandLineDelay
		config.Recursive = &commandLineRecursive
		config.DotFiles = &commandLineDotFiles
		cmd, cmdArgs := parseCommandString(commandLineCommand)
		config.Cmd = cmd
		config.CmdArgs = cmdArgs
		config.PidFile = commandLinePidFile
	case commandLineConfig != "":
		config.Load(commandLineConfig)
	}

	tasks, err := config.Tasks()

	if err != nil {
		log.Fatal(err)
	}

	done := make(chan os.Signal, 1)

	signal.Notify(done, os.Interrupt, os.Kill)

	for _, task := range *tasks {
		go task.Run()
	}

	for {
		select {
		case <-done:
			for _, task := range *tasks {
				task.Shutdown()
			}
			return
		}
	}

}
