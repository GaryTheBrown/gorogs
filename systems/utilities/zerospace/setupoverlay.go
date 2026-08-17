package zerospace

import (
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"os"

	"golang.org/x/sys/unix"
)

func (s *Struct) setupOverlay() error {
	const (
		sourceDir  = "/srv"
		targetDir  = "/share"
		tmpfsBase  = "/tmp/overlay-base"
		tmpfsUpper = tmpfsBase + "/upper"
		workDir    = tmpfsBase + "/work"
	)

	logger.InfoAppend(s.Name(), "[mkdir tmpfs base]")
	if err := os.MkdirAll(tmpfsBase, 0755); err != nil {
		return fmt.Errorf("failed to create tmpfs base directory: %w", err)
	}

	logger.InfoAppend(s.Name(), "[mount tempfs]")
	err := unix.Mount("tmpfs", tmpfsBase, "tmpfs", 0, "size=4k")
	if err != nil {
		return fmt.Errorf("failed to initialize tmpfs base: %w", err)
	}

	logger.InfoAppend(s.Name(), "[mkdir upper]")
	if err := os.MkdirAll(tmpfsUpper, 0755); err != nil {
		return fmt.Errorf("failed to create upper tracking directory: %w", err)
	}
	logger.InfoAppend(s.Name(), "[mkdir work]")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work tracking directory: %w", err)
	}

	logger.InfoAppend(s.Name(), "[mkdir target]")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target export directory: %w", err)
	}

	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", sourceDir, tmpfsUpper, workDir)
	logger.InfoAppend(s.Name(), "[mount overlay]")
	err = unix.Mount("overlay", targetDir, "overlay", 0, opts)
	if err != nil {
		return fmt.Errorf("failed to finalize live OverlayFS allocation on %s: %w", targetDir, err)
	}

	logger.InfoAppend(s.Name(), "[update share root]")
	config.ShareRoot = targetDir

	return nil
}
