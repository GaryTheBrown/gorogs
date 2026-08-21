package samba

import (
	"context"
	"gorogs/config"
	"gorogs/logger"
	"path/filepath"
	"regexp"

	"github.com/fsnotify/fsnotify"
)

func (s *Struct) startFSEventDirectoryWatcher(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error(s.Name(), "Failed to initialize fsnotify monitor subsystem hook", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(config.ShareRoot); err != nil {
		logger.ErrorF(s.Name(), "Failed to register directory target inside fsnotify monitor tracking path: %s", err, config.ShareRoot)
		return
	}

	logger.InfoF(s.Name(), "Live FS event-driven tracking loop successfully online monitoring path: %s", config.ShareRoot)

	validShareName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s()]+$`)

	for {
		select {
		case <-ctx.Done():
			logger.Debug(s.Name(), "Live share tracking event loop terminated cleanly.")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			shareName := filepath.Base(event.Name)

			if !validShareName.MatchString(shareName) || shareName == "nfs" || shareName == "ganesha" {
				continue
			}

			if event.Has(fsnotify.Create) {
				logger.InfoF(s.Name(), "Detected NEW folder creation: [%s]. Injecting live registry transaction...", shareName)
				if err := s.executeRegistryAdd(shareName, event.Name); err != nil {
					logger.ErrorF(s.Name(), "Dynamic registry hot injection transaction failed: %v", err, err.Error())
				} else {
					logger.InfoF(s.Name(), "Share [%s] is now live instantly with zero downtime.", shareName)
				}
			}

			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				logger.InfoF(s.Name(), "Detected folder extraction or relocation: [%s]. Removing memory key allocation...", shareName)
				if err := s.executeRegistryDelete(shareName); err != nil {
					logger.ErrorF(s.Name(), "Dynamic memory deletion transaction failed: %v", err, err.Error())
				} else {
					logger.InfoF(s.Name(), "Share [%s] successfully purged from live allocation grid.", shareName)
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error(s.Name(), "Inbound filesystem monitor tracking pipeline encountered an asynchronous operation fault", err)
		}
	}
}
