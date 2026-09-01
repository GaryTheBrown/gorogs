package main

import (
	"context"
	"os"
	"path/filepath"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/plugins/share/samba/structs"
	"gorogs/plugins/share/samba/vars"

	"github.com/fsnotify/fsnotify"
)

func (s *Struct) startFSEventCommentWatcher(ctx context.Context) {
	cWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.ErrorF(Name, "Failed to initialize metadata file comment watcher context: %w", err)
		return
	}
	s.commentWatcher = cWatcher
	defer s.commentWatcher.Close()

	if entries, err := os.ReadDir(config.ConstOriginalShareRoot); err == nil {
		for _, entry := range entries {
			if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
				fullPath := filepath.Join(config.ConstOriginalShareRoot, entry.Name())
				_ = s.commentWatcher.Add(fullPath)
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return

		case event, ok := <-s.commentWatcher.Events:
			if !ok {
				return
			}

			if filepath.Base(event.Name) != ".comment" {
				continue
			}

			shareName := filepath.Base(filepath.Dir(event.Name))

			switch {
			case event.Has(fsnotify.Create), event.Has(fsnotify.Write):
				comment := structs.ReadCommentFile(filepath.Dir(event.Name))
				_ = s.sys.NotifyCommentUpdate(shareName, comment)

			case event.Has(fsnotify.Remove), event.Has(fsnotify.Rename):
				_ = s.sys.NotifyCommentUpdate(shareName, vars.DefaultShareComment)
			}

		case err, ok := <-s.commentWatcher.Errors:
			if !ok {
				return
			}
			logger.Error(Name, "Metadata comment tracker pipe encountered an asynchronous operation fault", err)
		}
	}
}
