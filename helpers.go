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

func isDotfile(path string) bool {
	return dotFileRx.MatchString(filepath.Base(path))
}
