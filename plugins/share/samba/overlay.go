package main

import (
	"gorogs/logger"
	"gorogs/plugins/share/samba/vars"
	"os"
	"syscall"
)

func (s *Struct) setupOverlay() error {
	logger.Debug(Name, "[OVERLAY:MKDIR BASE]")
	if err := os.MkdirAll(vars.SambaBaseLibDir, 0755); err != nil {
		return err
	}
	logger.Debug(Name, "[OVERLAY: MOUNT TMPFS]")
	if err := syscall.Mount("tmpfs", vars.SambaBaseLibDir, "tmpfs", 0, "size=256M,mode=1777"); err != nil {
		return err
	}
	logger.Debug(Name, "[OVERLAY:MKDIR PRIVATE]")
	return os.MkdirAll(vars.InternalDBPath, 0755)
}
