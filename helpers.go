package main

import (
	"github.com/howeyc/fsnotify"
	"log"
	"os"
	"path/filepath"
	"strings"
)

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
	log.Printf("Define watcher %s\n", dir)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Print(err)
		return
	}

	defer log.Printf("Remove watcher %s\n", dir)
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
