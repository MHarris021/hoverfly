package main

import (
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

type RecursiveWatcher struct {
	watcher *fsnotify.Watcher
	paths   map[string]bool
}

func NewRecursiveWatcher(path string, recursive bool) (*RecursiveWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	rw := &RecursiveWatcher{watcher, make(map[string]bool)}
	if recursive {
		if err = rw.AddPath(path); err != nil {
			return nil, err
		}
	} else {
		if err := rw.watcher.Add(path); err != nil {
			return nil, err
		}
		rw.paths[path] = true
	}
	return rw, nil
}

func (rw *RecursiveWatcher) Close() error {
	return rw.watcher.Close()
}

func (rw *RecursiveWatcher) AddPath(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsDir() {
			if !validDir(info) || rw.paths[path] {
				return filepath.SkipDir
			}
			if err := rw.watcher.Add(path); err != nil {
				return err
			}
			rw.paths[path] = true
		}
		return nil
	})
}

func (rw *RecursiveWatcher) Run() chan string {
	files := make(chan string, 10)
	go func() {
		for rw.watcher.Events != nil && rw.watcher.Errors != nil {
			select {
			case event, ok := <-rw.watcher.Events:
				if !ok {
					rw.watcher.Events = nil
					break
				}
				if info, err := os.Stat(event.Name); err == nil {
					if validDir(info) {
						if event.Op&fsnotify.Create == fsnotify.Create {
							if err := rw.AddPath(event.Name); err != nil {
								Error.Println(err)
							}
						}
					} else if validFile(info) {
						if event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Write == fsnotify.Write {
							files <- event.Name
						}
					}
				}
			case err, ok := <-rw.watcher.Errors:
				if !ok {
					rw.watcher.Errors = nil
					break
				}
				Error.Println(err)
			}
		}
		close(files)
	}()
	return files
}
