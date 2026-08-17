package samba

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
	"gorogs/systems/systeminterface"

	"github.com/fsnotify/fsnotify"
)

const (
	Name       = "Samba"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

type Struct struct {
	sState      systeminterface.SysStateEnum
	cmd         *exec.Cmd
	readyChan   chan struct{}
	cancelWatch context.CancelFunc
}

func (_ *Struct) Name() string                               { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum       { return Type }
func (_ *Struct) IsCritical() bool                           { return IsCritical }
func (_ *Struct) AutoStart() bool                            { return AutoStart }
func (s *Struct) State(in systeminterface.SysStateEnum) bool { return s.sState == in }

func (s *Struct) Setup() {
	logger.Info(s.Name(), "Commencing pre-flight checks and configuration layout parsing...")

	if err := s.writeMasterSambaConfig(config.Hostname); err != nil {
		logger.Fatal(s.Name(), "failed to execute master config write utility", err)
	}

	if err := s.writeDynamicSharesConfig(); err != nil {
		logger.Fatal(s.Name(), "failed to execute dynamic shares configuration layout write utility", err)
	}

	logger.Info(s.Name(), "Configuration pre-flight generation phase successfully completed.")
	s.sState = systeminterface.SETUP
}

func (s *Struct) writeMasterSambaConfig(serverName string) error {
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
		"    dns proxy = no\n" +
		"    hostname lookups = no\n" +
		"\n" +
		"    include = /dev/shm/smb-shares.conf\n"

	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (s *Struct) writeDynamicSharesConfig() error {
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
				logger.ErrorF(s.Name(), "Validation Error: Share directory name [%s] contains illegal characters and was dropped.", nil, entry.Name())
				continue
			}

			if entry.Name() == "nfs" || entry.Name() == "ganesha" {
				if logger.DebugActive {
					logger.DebugF(s.Name(), "Bypassing reserved sub-directory name from Samba compilation: %s", entry.Name())
				}
				continue
			}

			fullPath := filepath.Join(config.ShareRoot, entry.Name())
			if logger.DebugActive {
				logger.Debug(s.Name(), fmt.Sprintf("Compiling export allocation: [%s] mapping to physical path %s", entry.Name(), fullPath))
			}

			fmt.Fprintf(file, "\n[%s]\n", entry.Name())
			fmt.Fprintf(file, "    path = %s\n", fullPath)
			fmt.Fprintf(file, "    browseable = yes\n")
			fmt.Fprintf(file, "    read only = yes\n")
			fmt.Fprintf(file, "    guest ok = yes\n")
		}
	}
	return nil
}

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Spawning primary Samba smbd background engine...")

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

	logger.InfoF(s.Name(), "Samba background engine active under operational Process ID: %d. Waiting for socket readiness...", s.cmd.Process.Pid)

	select {
	case <-s.readyChan:
		logger.Info(s.Name(), "Samba successfully bound network ports and is accepting incoming client requests.")

		if !config.IsDisabled("livechanges") {
			watchCtx, cancel := context.WithCancel(context.Background())
			s.cancelWatch = cancel
			go s.startFSEventDirectoryWatcher(watchCtx)
		} else {
			logger.Info(s.Name(), "Live share adjustments are deactivated via system configuration flags.")
		}
		s.sState = systeminterface.STARTED
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for Samba daemon process tree to declare socket readiness")
	}
}

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

	var debounceTimer *time.Timer
	const debounceDuration = 250 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			logger.Debug(s.Name(), "Live share tracking event loop terminated cleanly.")
			return

		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			if event.Has(fsnotify.Create) || event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				logger.DebugF(s.Name(), "FS Event intercepted -> Action: %s on Path: %s", event.Op.String(), event.Name)

				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDuration, func() {
					logger.Info(s.Name(), "FSEvent stabilization window cleared. Recompiling dynamic shares...")

					if err := s.writeDynamicSharesConfig(); err != nil {
						logger.Error(s.Name(), "Failed to compile updated share text block modifications to memory configuration space", err)
						return
					}

					if s.cmd != nil && s.cmd.Process != nil {
						if err := s.cmd.Process.Signal(syscall.SIGHUP); err != nil {
							logger.Error(s.Name(), "Failed to transmit SIGHUP configuration hot-reload trigger command to smbd process", err)
						} else {
							logger.Info(s.Name(), "Samba hot-reload signal executed successfully. Share changes are live with zero client downtime.")
						}
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			logger.Error(s.Name(), "Inbound filesystem monitor tracking pipeline encountered an asynchronous operation fault", err)
		}
	}
}

func (s *Struct) streamSubsystemLogs(pipe io.ReadCloser) {
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

		if logger.IsDebugActive(s.Name()) {
			logger.Debug(s.Name(), trimmedLine)
		} else {
			logger.Info(s.Name(), trimmedLine)
		}
	}

	if !hasSignaledReady {
		close(s.readyChan)
	}

	if err := scanner.Err(); err != nil {
		logger.Error(s.Name(), "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (s *Struct) Stop() {
	if s.cancelWatch != nil {
		s.cancelWatch()
	}

	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	logger.Info(s.Name(), "Initiating graceful termination sequence on Samba daemon threads...")
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		s.cmd.Process.Kill()
		return
	}
	_ = s.cmd.Wait()
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("samba background system execution tracking instance is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
