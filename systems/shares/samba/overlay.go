package samba

import (
	"gorogs/logger"
	"gorogs/systems/shares/samba/vars"
	"os"
	"syscall"
)

func (s *Struct) setupOverlay() error {
	logger.DebugAppend(Name, "[OVERLAY:MKDIR BASE]")
	if err := os.MkdirAll(vars.SambaBaseLibDir, 0755); err != nil {
		return err
	}
	logger.DebugAppend(Name, "[OVERLAY: MOUNT TMPFS]")
	if err := syscall.Mount("tmpfs", vars.SambaBaseLibDir, "tmpfs", 0, "size=256M,mode=1777"); err != nil {
		return err
	}
	logger.DebugAppend(Name, "[OVERLAY:MKDIR PRIVATE]")
	return os.MkdirAll(vars.InternalDBPath, 0755)
}
