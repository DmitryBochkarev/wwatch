package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
)

const (
	DEFAULT_DIR           = "."
	DEFAULT_CWD           = "."
	DEFAULT_MATCH_PATTERN = ".*"
	DEFAULT_DELAY         = "100ms"
	DEFAULT_RECURSIVE     = false
	DEFAULT_DOTFILES      = false
	DEFAULT_AFTER_CHANGE  = false
)

var (
	commandLineDir, commandLineCwd, commandLineMatchPattern, commandLineExt, commandLineIgnorePattern string
	commandLineAfterChange                                                                            bool
	commandLineDelay                                                                                  string
	commandLineCommand, commandLineOnStartCommand, commandLinePidFile                                 string
	commandLineConfig                                                                                 string
	commandLineRecursive, commandLineDotFiles, commandLinePrintVersion                                bool

	config        Config
	tasks         *map[string]*Task
	outletFactory = NewOutletFactory()
)

func init() {
	flag.StringVar(&commandLineDir, "dir", DEFAULT_DIR, "directory to watch")
	flag.StringVar(&commandLineCwd, "cwd", DEFAULT_CWD, "current working directory")
	flag.StringVar(&commandLineMatchPattern, "match", DEFAULT_MATCH_PATTERN, "file(fullpath) match regexp")
	flag.StringVar(&commandLineExt, "ext", "", "extentions of files to watch: -ext='less,js,coffee'")
	flag.StringVar(&commandLineIgnorePattern, "ignore", "", "regexp patter for ignore watch")
	flag.BoolVar(&commandLineAfterChange, "after", DEFAULT_AFTER_CHANGE, "run command only after files changed")
	flag.StringVar(&commandLineDelay, "delay", DEFAULT_DELAY, "delay before rerun cmd")
	flag.StringVar(&commandLineCommand, "cmd", "", "command to run, rerun on file changed")
	flag.StringVar(&commandLineOnStartCommand, "onstart", "", "command to run on start")
	flag.StringVar(&commandLinePidFile, "pidfile", "", "file that content pid of running process")
	flag.StringVar(&commandLineConfig, "config", "", "path to configuration file(*.toml)")
	flag.BoolVar(&commandLineRecursive, "recursive", DEFAULT_RECURSIVE, "walk recursive over directories")
	flag.BoolVar(&commandLineDotFiles, "dotfiles", DEFAULT_DOTFILES, "watch on dotfiles")
	flag.BoolVar(&commandLinePrintVersion, "version", false, "print version")
}

func main() {
	flag.Parse()

	log.SetOutput(outletFactory)
	if commandLinePrintVersion {
		log.Fatalf("version: %s", VERSION)
	}

	switch {
	case commandLineCommand == "" && commandLineConfig == "":
		log.Fatal(errors.New("You should specify command or path to configuration file"))
	case commandLineCommand != "":
		config.Dir = commandLineDir
		config.Cwd = commandLineCwd
		config.Match = commandLineMatchPattern

		config.Ext = strings.Split(strings.Replace(commandLineExt, " ", "", -1), ",")
		config.After = &commandLineAfterChange
		config.Delay = commandLineDelay
		config.Recursive = &commandLineRecursive
		config.DotFiles = &commandLineDotFiles

		onStartCmd, onStartCmdArgs := parseCommandString(commandLineOnStartCommand)
		config.OnStartCmd = onStartCmd
		config.OnStartCmdArgs = onStartCmdArgs

		cmd, cmdArgs := parseCommandString(commandLineCommand)
		config.Cmd = cmd
		config.CmdArgs = cmdArgs

		config.PidFile = commandLinePidFile
	case commandLineConfig != "":
		configDir, err := filepath.Abs(filepath.Dir(commandLineConfig))
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("change working directory to %s", configDir)
		if err := os.Chdir(configDir); err != nil {
			log.Fatal(err)
		}

		config.Load(commandLineConfig)
	}

	tasks, err := config.Tasks(outletFactory)
	if err != nil {
		log.Fatal(err)
	}

	if _, mainSection := (*tasks)[""]; config.OnStartCmd != "" && !mainSection {
		exe := os.Expand(config.OnStartCmd, os.Getenv)
		var args = make([]string, len(config.OnStartCmdArgs))
		for i, arg := range config.OnStartCmdArgs {
			args[i] = os.Expand(arg, os.Getenv)
		}

		log.Printf("run main onstart command %s %v\n", exe, strings.Join(args, " "))
		command := exec.Command(exe, args...)
		command.Dir = config.Cwd
		command.Stdout = outletFactory
		command.Stderr = outletFactory
		command.Run()
	}

	for _, task := range *tasks {
		go task.Run()
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, os.Kill)

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
