package main

import (
	"context"
	"gorogs/config"
	"gorogs/logger"
	"os"
	"path/filepath"
	"regexp"

	"github.com/fsnotify/fsnotify"
)

func (s *Struct) startFSEventDirectoryWatcher(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.ErrorF(Name, "Failed to initialize fsnotify monitor subsystem hook: %w", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(config.ShareRoot); err != nil {
		logger.ErrorF(Name, "Failed to register directory target inside fsnotify monitor tracking path: %w", err)
		return
	}

	validShareName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s()]+$`)

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			shareName := filepath.Base(event.Name)
			if !validShareName.MatchString(shareName) {
				continue
			}
			if event.Has(fsnotify.Create) {
				if info, err := os.Stat(config.ShareRoot); err == nil && info.IsDir() {
					if err := s.sys.NotifyCreate(shareName, config.ShareRoot); err == nil {
						if s.commentWatcher != nil {
							_ = s.commentWatcher.Add(event.Name)
						}
					}
				}
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				if err := s.sys.NotifyRemove(shareName); err != nil {

				}
				if s.commentWatcher != nil {
					_ = s.commentWatcher.Remove(event.Name)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error(Name, "Inbound filesystem monitor tracking pipeline encountered an asynchronous operation fault", err)
		}
	}
}
