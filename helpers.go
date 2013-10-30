package main

import (
	"github.com/howeyc/fsnotify"
	"log"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	dotFileRx = regexp.MustCompile(`^\..*$`)
)

func startWatcher(dir string, event chan *fsnotify.FileEvent) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Watch(dir)

	if err != nil {
		return nil, err
	}

	go func() {
		for {
			select {
			case ev, ok := <-watcher.Event:
				if !ok {
					return
				}
				event <- ev
			case err, ok := <-watcher.Error:
				if !ok {
					return
				}
				log.Println("watch error: ", err)
			}
		}
	}()

	return watcher, nil
}

func parseCommandString(commandString string) []string {
	cmdargs := strings.SplitN(commandString, " ", -1)
	if len(cmdargs) > 0 && cmdargs[0] == "" {
		return []string{}
	}
	return cmdargs
}

func isDotfile(path string) bool {
	return dotFileRx.MatchString(filepath.Base(path))
}
