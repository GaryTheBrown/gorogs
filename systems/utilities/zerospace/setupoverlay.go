package zerospace

import (
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"os"

	"golang.org/x/sys/unix"
)

func (m *ZeroSpaceStruct) setupOverlay() error {
	const (
		sourceDir  = "/srv"
		targetDir  = "/share"
		tmpfsUpper = "/tmp/node"
		workDir    = "/tmp/work"
	)
	logger.InfoAppend(m.Name(), "[mkdir upper]")
	if err := os.MkdirAll(tmpfsUpper, 0755); err != nil {
		return fmt.Errorf("failed to create upper tracking directory: %w", err)
	}
	logger.InfoAppend(m.Name(), "[mkdir work]")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work tracking directory: %w", err)
	}
	logger.InfoAppend(m.Name(), "[mkdir target]")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target export directory: %w", err)
	}

	logger.InfoAppend(m.Name(), "[mount tempfs]")
	err := unix.Mount("tmpfs", tmpfsUpper, "tmpfs", 0, "size=4k,nr_inodes=0")
	if err != nil {
		return fmt.Errorf("failed to initialize zero-inode upper space mask: %w", err)
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", sourceDir, tmpfsUpper, workDir)
	logger.InfoAppend(m.Name(), "[mount overlay]")
	err = unix.Mount("overlay", targetDir, "overlay", 0, opts)
	if err != nil {
		return fmt.Errorf("failed to finalize live OverlayFS allocation on %s: %w", targetDir, err)
	}
	logger.InfoAppend(m.Name(), "[update share root]")
	config.ShareRoot = targetDir

	return nil
}
