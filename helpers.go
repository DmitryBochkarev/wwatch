package main

import (
	"github.com/howeyc/fsnotify"
	"log"
	"path/filepath"
	"strings"
)

func startWatcher(dir string, quit chan bool, event chan *fsnotify.FileEvent) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
		return
	}

	err = watcher.Watch(dir)

	if err != nil {
		log.Print(err)
		return
	}

	go func() {
		defer watcher.Close()
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
	}()

}

func parseCommandString(commandString string) (exe string, args []string) {
	cmdSplit := strings.SplitN(commandString, " ", 2)
	exe = cmdSplit[0]
	if len(cmdSplit) > 1 {
		args = strings.Split(cmdSplit[1], " ")
	}
	return
}

func resolvePath(start string, parts ...string) (path string) {
	path = start
	for _, part := range parts {
		if filepath.IsAbs(part) {
			path = part
			continue
		}
		path = filepath.Join(path, part)
	}
	return
}
