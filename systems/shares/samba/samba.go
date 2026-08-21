package samba

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "Samba"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

var (
	programPath      = "/usr/bin/smbd"
	masterConfigPath = "/etc/samba/smb.conf"
	netPath          = "/usr/bin/net"
	smbpasswdPath    = "/usr/bin/smbpasswd"
	sambaBaseLibDir  = "/var/lib/samba"
	internalDBPath   = sambaBaseLibDir + "/private"
	internalDBFile   = sambaBaseLibDir + "/registry.tdb"
	registryTxtPath  = sambaBaseLibDir + "/registry_import.txt"
)

type Struct struct {
	sState      systeminterface.SysStateEnum
	cmd         *exec.Cmd
	logWriter   *helpers.SubsystemWriter
	readyChan   chan struct{}
	cancelWatch context.CancelFunc
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	logger.Info(s.Name(), "Commencing high-speed RAM-backed storage appliance pre-flight validation...")

	// 1. KERNEL RAM-DISK MOUNT PASS
	if err := os.MkdirAll(sambaBaseLibDir, 0755); err != nil {
		logger.Fatal(s.Name(), "Failed to pre-stage native library tracking folder directory", err)
	}
	if err := syscall.Mount("tmpfs", sambaBaseLibDir, "tmpfs", 0, "size=256M,mode=1777"); err != nil {
		logger.Fatal(s.Name(), "KERNEL MOUNT PANIC: Failed to allocate high-speed RAM layer", err)
	}

	// 2. RAM INTERNALS INITIALIZATION
	if err := os.MkdirAll(internalDBPath, 0755); err != nil {
		logger.Fatal(s.Name(), "Failed to configure nested runtime state pools inside mounted RAM namespace", err)
	}

	// First, write the master configuration text template file so sub-utilities have a valid context
	if err := s.writeMasterSambaConfig(); err != nil {
		logger.Fatal(s.Name(), "failed to execute master config write utility", err)
	}

	// 3. SECURE SYSTEM STATE TRACKING DATABASES
	logger.Info(s.Name(), "Initializing structured local user policy tracking databases...")
	_ = os.WriteFile(filepath.Join(sambaBaseLibDir, "account_policy.tdb"), []byte{}, 0600)
	_ = os.WriteFile(filepath.Join(sambaBaseLibDir, "winbindd_idmap.tdb"), []byte{}, 0600)

	// Build passdb schema structures securely using the valid text file context
	cmdPasswd := exec.Command(smbpasswdPath, "-L", "-c", masterConfigPath, "-a", "nobody", "-n")
	if output, err := cmdPasswd.CombinedOutput(); err != nil {
		logger.ErrorF(s.Name(), "User database initialization failed: %s ERROR: %v", err, strings.TrimSpace(string(output)), err.Error())
	} else {
		logger.Info(s.Name(), "Local user policy tracking database successfully pre-seeded.")
	}

	logger.Info(s.Name(), "Generating independent machine security identifier tokens...")
	cmdSID := exec.Command(netPath, "setlocalsid", "S-1-5-21-1111111111-2222222222-3333333333", "-s", masterConfigPath)
	if output, err := cmdSID.CombinedOutput(); err != nil {
		logger.ErrorF(s.Name(), "Failed to register machine identity tokens: %s ERROR: %v", err, strings.TrimSpace(string(output)), err.Error())
	}

	// Compile and inject your entire registry configuration blocks straight to disk
	logger.Info(s.Name(), "Pre-seeding global parameters and movie share blocks into RAM database...")
	s.injectAllSharesToRegistry()

	logger.Info(s.Name(), "Samba configuration pre-flight generation phase successfully completed.")
	s.sState = systeminterface.SETUP
}

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Spawning primary Samba smbd background engine...")

	args := []string{"--foreground", "--no-process-group", "-s", masterConfigPath, "--debug-stdout"}
	var binaryPath string

	if logger.IsDebugActive(s.Name()) {
		binaryPath = "/usr/bin/strace"
		// -f traces child forks; -e trace limits output to file-system/socket tracking
		args = []string{"-f", "-e", "trace=openat,stat,connect,socket", programPath, "--foreground", "--no-process-group", "-s", masterConfigPath, "--debug-stdout", "-d", "3"}

		// args = append(args, "-d", "3")
	} else {
		args = append(args, "-d", "0")
	}

	// s.cmd = exec.Command(programPath, args...)
	s.cmd = exec.Command(binaryPath, args...)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if logger.IsDebugActive(s.Name()) {
		s.logWriter = helpers.NewSubsystemWriter(s.Name(), nil, nil, nil)
		s.cmd.Stdout = s.logWriter
		s.cmd.Stderr = s.logWriter
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background smbd process: %w", err)
	}

	logger.InfoF(s.Name(), "Samba active (PID: %d). Probing port 445...", s.cmd.Process.Pid)

	portBound := false
	maxWait := 20 * time.Second
	currentTick := 100 * time.Millisecond
	startTime := time.Now()
	probeAttempts := 0

	for time.Since(startTime) < maxWait {
		probeAttempts++
		if err := s.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			if s.logWriter != nil {
				s.logWriter.Close()
			}
			return fmt.Errorf("samba smbd daemon process terminated unexpectedly during boot")
		}
		if helpers.WaitForSocket("tcp", "127.0.0.1:445", 50*time.Millisecond) {
			portBound = true
			break
		}
		if probeAttempts > 10 {
			currentTick = time.Duration(float64(currentTick) * 1.5)
			if currentTick > 2*time.Second {
				currentTick = 2 * time.Second
			}
		}
		time.Sleep(currentTick)
	}

	if !portBound {
		if s.logWriter != nil {
			s.logWriter.Close()
		}
		return fmt.Errorf("timeout reached waiting for Samba daemon to bind network port 445")
	}

	logger.Info(s.Name(), "Samba successfully bound network ports")

	if !config.IsDisabled("livechanges") {
		watchCtx, cancel := context.WithCancel(context.Background())
		s.cancelWatch = cancel
		go s.startFSEventDirectoryWatcher(watchCtx)
	}

	s.sState = systeminterface.STARTED
	return nil
}

func (s *Struct) Stop() {
	if s.cancelWatch != nil {
		s.cancelWatch()
	}

	if s.cmd != nil && s.cmd.Process != nil {
		logger.Info(s.Name(), "Initiating graceful termination sequence on Samba daemon threads...")

		if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.cmd.Process.Kill()
		}

		_ = s.cmd.Wait()
		logger.Info(s.Name(), "Samba primary background daemon thread terminated cleanly.")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
	}

	logger.InfoF(s.Name(), "Unmounting memory partition allocation layer cleanly: %s", sambaBaseLibDir)
	if err := syscall.Unmount(sambaBaseLibDir, 0); err != nil {
		logger.ErrorF(s.Name(), "Failed to unmount memory partition layer cleanly from layout ERROR: %v", err, err.Error())
	}

	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("samba background system execution tracking instance is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
