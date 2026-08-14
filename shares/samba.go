package shares

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"

	"github.com/fsnotify/fsnotify"
)

type SambaShare struct {
	cmd         *exec.Cmd
	readyChan   chan struct{}      // Synchronises Go lifecycle with Samba state
	cancelWatch context.CancelFunc // Cleans up dynamic directory tracking threads on Stop
}

func (s *SambaShare) Setup() error {
	logger.Info("SAMBA", "Commencing pre-flight checks and configuration layout parsing...")

	if err := s.writeMasterSambaConfig(config.Name); err != nil {
		return fmt.Errorf("failed to execute master config write utility: %w", err)
	}

	if err := s.writeDynamicSharesConfig(); err != nil {
		return fmt.Errorf("failed to execute dynamic shares configuration layout write utility: %w", err)
	}

	logger.Info("SAMBA", "Configuration pre-flight generation phase successfully completed.")
	return nil
}

func (s *SambaShare) writeMasterSambaConfig(serverName string) error {
	masterConfigPath := "/etc/samba/smbd.conf"

	masterContent := "[global]\n" +
		"    netbios name = " + serverName + "\n" +
		"    server string = Read only Share\n" +
		"    log file = /var/log/samba/log.%%m\n" +
		"    max log size = 1000\n" +
		"    logging = file\n" +
		"    server role = standalone server\n" +
		"    map to guest = bad user\n" +
		"    usershare allow guests = yes\n" +
		"    usershare max shares = 0\n" +
		"\n" +
		"    include = /dev/shm/smb-shares.conf\n"

	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (s *SambaShare) writeDynamicSharesConfig() error {
	shareConfigPath := "/dev/shm/smb-shares.conf"
	file, err := os.Create(shareConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()

	entries, err := os.ReadDir(config.ShareRoot)
	if err != nil {
		return err
	}

	validShareName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s()]+$`)

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "." && entry.Name() != ".." {
			if !validShareName.MatchString(entry.Name()) {
				logger.Error("SAMBA", fmt.Sprintf("Validation Error: Share directory name [%s] contains illegal characters and was dropped.", entry.Name()), nil)
				continue
			}

			if entry.Name() == "nfs" || entry.Name() == "ganesha" {
				logger.Debug("SAMBA", fmt.Sprintf("Bypassing reserved sub-directory name from Samba compilation: %s", entry.Name()))
				continue
			}

			fullPath := filepath.Join(config.ShareRoot, entry.Name())
			logger.Debug("SAMBA", fmt.Sprintf("Compiling export allocation: [%s] mapping to physical path %s", entry.Name(), fullPath))

			fmt.Fprintf(file, "\n[%s]\n", entry.Name())
			fmt.Fprintf(file, "    path = %s\n", fullPath)
			fmt.Fprintf(file, "    browseable = yes\n")
			fmt.Fprintf(file, "    read only = yes\n")
			fmt.Fprintf(file, "    guest ok = yes\n")
		}
	}
	return nil
}

func (s *SambaShare) Start() error {
	logger.Info("SAMBA", "Spawning primary Samba smbd background engine...")

	s.readyChan = make(chan struct{})

	s.cmd = exec.Command("/usr/sbin/smbd", "--foreground", "--no-process-group", "--debug-stdout", "-s", "/etc/samba/smbd.conf")
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	sambaPipe, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link Samba stdout processing pipeline: %w", err)
	}
	s.cmd.Stderr = s.cmd.Stdout

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background smbd execution tracking loop: %w", err)
	}

	go s.streamSubsystemLogs(sambaPipe)

	logger.Info("SAMBA", fmt.Sprintf("Samba background engine active under operational Process ID: %d. Waiting for socket readiness...", s.cmd.Process.Pid))

	select {
	case <-s.readyChan:
		logger.Info("SAMBA", "Samba successfully bound network ports and is accepting incoming client requests.")

		// LIVE CHANGES PIPELINE CONFIGURATION CHECK
		if config.LiveChangesEnabled {
			watchCtx, cancel := context.WithCancel(context.Background())
			s.cancelWatch = cancel
			go s.startFSEventDirectoryWatcher(watchCtx)
		} else {
			logger.Info("SAMBA", "Live share adjustments are deactivated via system configuration flags.")
		}

		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for Samba daemon process tree to declare socket readiness")
	}
}

func (s *SambaShare) startFSEventDirectoryWatcher(ctx context.Context) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logger.Error("SAMBA", "Failed to initialize fsnotify monitor subsystem hook", err)
		return
	}
	defer watcher.Close()

	if err := watcher.Add(config.ShareRoot); err != nil {
		logger.Error("SAMBA", "Failed to register directory target inside fsnotify monitor tracking path: "+config.ShareRoot, err)
		return
	}

	logger.Info("SAMBA", "Live FS event-driven tracking loop successfully online monitoring path: "+config.ShareRoot)

	var debounceTimer *time.Timer
	const debounceDuration = 250 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			logger.Debug("SAMBA", "Live share tracking event loop terminated cleanly.")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Capture directory changes: creations, removals, and renames
			if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				logger.Debug("SAMBA", fmt.Sprintf("FS Event intercepted -> Action: %s on Path: %s", event.Op.String(), event.Name))

				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				// Reset the rolling debouncer execution latch timer
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					logger.Info("SAMBA", "FSEvent stabilization window cleared. Recompiling dynamic shares...")

					// 1. Rebuild the dynamic shares configuration tracking file
					if err := s.writeDynamicSharesConfig(); err != nil {
						logger.Error("SAMBA", "Failed to compile updated share text block modifications to memory configuration space", err)
						return
					}

					// 2. Dispatch a SIGHUP signal directly to smbd to trigger a seamless hot-reload
					if s.cmd != nil && s.cmd.Process != nil {
						if err := s.cmd.Process.Signal(syscall.SIGHUP); err != nil {
							logger.Error("SAMBA", "Failed to transmit SIGHUP configuration hot-reload trigger command to smbd process", err)
						} else {
							logger.Info("SAMBA", "Samba hot-reload signal executed successfully. Share changes are live with zero client downtime.")
						}
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error("SAMBA", "Inbound filesystem monitor tracking pipeline encountered an asynchronous operation fault", err)
		}
	}
}

func (s *SambaShare) streamSubsystemLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	hasSignaledReady := false

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if !hasSignaledReady && (strings.Contains(line, "started") || strings.Contains(line, "Ready for connections") || strings.Contains(line, "smbd_open_once_socket")) {
			close(s.readyChan)
			hasSignaledReady = true
		}

		if logger.IsDebugActive("samba") {
			logger.Debug("SAMBA", trimmedLine)
		} else {
			logger.Info("SAMBA", trimmedLine)
		}
	}

	if !hasSignaledReady {
		close(s.readyChan)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("SAMBA", "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (s *SambaShare) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("samba background system execution tracking instance is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}

func (s *SambaShare) IsCritical() bool { return true }

func (s *SambaShare) Stop() error {
	// Cancel the active background fsnotify watcher context cleanly
	if s.cancelWatch != nil {
		s.cancelWatch()
	}

	if s.cmd == nil || s.cmd.Process == nil {
		return nil
	}
	logger.Info("SAMBA", "Initiating graceful termination sequence on Samba daemon threads...")
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return s.cmd.Process.Kill()
	}
	_ = s.cmd.Wait()
	return nil
}
