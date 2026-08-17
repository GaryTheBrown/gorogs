package netbios

import (
	"bufio"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/systeminterface"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	Name       = "NetBIOS"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = false
)

type Struct struct {
	sState systeminterface.SysStateEnum
	cmd    *exec.Cmd
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	if err := os.MkdirAll("/var/log/samba", 0755); err != nil {
		logger.Fatal(Name, "failed to provision local samba logging directories", err)
	}
	logger.DebugAppend(Name, "[mkdir]")

	if err := s.writeMasterNmbdConfig(config.Hostname); err != nil {
		logger.FatalF(Name, "failed to write %s config file", err, Name)
	}
	logger.DebugAppend(Name, "[write config]")

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) writeMasterNmbdConfig(serverName string) error {
	masterConfigPath := "/etc/samba/nmbd.conf"

	masterContent := "[global]\n" +
		"    netbios name = " + serverName + "\n" +
		"    workgroup = WORKGROUP\n" +
		"    server string = Network Discovery Beacon\n" +
		"    log file = /var/log/samba/log.nmbd\n" +
		"    max log size = 1000\n" +
		"    logging = file\n" +
		"    local master = no\n" +
		"    preferred master = no\n" +
		"    domain master = no\n"

	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")

	s.cmd = exec.Command("/usr/sbin/nmbd", "--foreground", "--no-process-group", "--debug-stdout", "-s", "/etc/samba/nmbd.conf")
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	nmbPipe, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link NetBIOS nmbd stdout processing pipeline: %w", err)
	}
	s.cmd.Stderr = s.cmd.Stdout

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start standalone nmbd beacon process: %w", err)
	}

	go s.streamSubsystemLogs(nmbPipe)

	logger.DebugF(Name, "NetBIOS tracker process running (PID: %d). Probing UDP socket...", s.cmd.Process.Pid)

	for range 30 {
		conn, err := net.DialTimeout("udp4", "127.0.0.1:137", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			logger.Debug(Name, "Verified: NetBIOS daemon has successfully bound UDP Port 137 and is online.")
			s.sState = systeminterface.STARTED
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for NetBIOS network interface to bind socket 137")
}

func (s *Struct) streamSubsystemLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		logger.Debug(Name, trimmedLine)
	}

	if err := scanner.Err(); err != nil {
		logger.Error(Name, "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (s *Struct) Stop() {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}
	logger.Debug(Name, "Initiating graceful termination sequence on NetBIOS beacon threads...")
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		s.cmd.Process.Kill()
		return
	}
	_ = s.cmd.Wait()
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("netbios background system execution tracking instance is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
