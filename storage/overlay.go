package storage

import (
	"fmt"
	"gorogs/config"
	"os"

	"golang.org/x/sys/unix"
)

// ShareRoot is now a variable that defaults to your original value.
// The setup function will update this to point to the masked folder path.

// SetupZeroSpaceOverlay configures a space-masked OverlayFS.
// It reads from the raw source (/srv), applies the zero-space mask,
// mounts it at /share, and updates the global ShareRoot variable.
func SetupZeroSpaceOverlay() error {
	const (
		sourceDir  = "/srv"
		targetDir  = "/share"
		tmpfsUpper = "/tmp/node"
		workDir    = "/tmp/work"
	)

	// 1. Ensure internal tracking and orchestration directories exist
	if err := os.MkdirAll(tmpfsUpper, 0755); err != nil {
		return fmt.Errorf("failed to create upper tracking directory: %w", err)
	}
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("failed to create work tracking directory: %w", err)
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target export directory: %w", err)
	}

	// 2. Mount a tiny 4KB tmpfs with exactly 0 allowed inodes.
	// This forces a permanent "No space left on device" kernel-level status.
	err := unix.Mount("tmpfs", tmpfsUpper, "tmpfs", 0, "size=4k,nr_inodes=0")
	if err != nil {
		return fmt.Errorf("failed to initialize zero-inode upper space mask: %w", err)
	}

	// 3. Construct the OverlayFS parameter line using /srv as our source lowerdir
	opts := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", sourceDir, tmpfsUpper, workDir)

	// 4. Mount the masked OverlayFS directly onto the clean /share directory
	err = unix.Mount("overlay", targetDir, "overlay", 0, opts)
	if err != nil {
		return fmt.Errorf("failed to finalize live OverlayFS allocation on %s: %w", targetDir, err)
	}

	// 5. Update the variable pointer so the rest of your app targets the masked path
	config.ShareRoot = targetDir

	return nil
}
