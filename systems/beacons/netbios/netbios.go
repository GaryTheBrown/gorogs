package netbios

import (
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/systeminterface"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	Name       = "NetBIOS"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = false
)

var (
	programPath      = "/usr/bin/nmbd"
	masterConfigPath = "/etc/nmbd.conf"
)

type Struct struct {
	sState    systeminterface.SysStateEnum
	cmd       *exec.Cmd
	logWriter *helpers.SubsystemWriter
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	if err := s.writeMasterNmbdConfig(config.Hostname); err != nil {
		logger.FatalF(Name, "failed to write %s config file", err, Name)
	}
	logger.DebugAppend(Name, "[write config]")

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) writeMasterNmbdConfig(serverName string) error {

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

	s.cmd = exec.Command(programPath, "--foreground", "--no-process-group", "--debug-stdout", "-s", masterConfigPath)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.DebugAppend(Name, "[CMD SETUP]")

	if logger.IsDebugActive(Name) {
		s.logWriter = helpers.NewSubsystemWriter(Name+".CMD", nil, nil, nil)
		s.cmd.Stdout = s.logWriter
		s.cmd.Stderr = s.logWriter
		logger.DebugAppend(Name, "[LINK STDOUT->LOG]")
	}

	if err := s.cmd.Start(); err != nil {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to start standalone nmbd beacon process: %w", err)
	}
	logger.DebugAppend(Name, "[CMD START]")

	if !helpers.WaitForSocket("udp4", "127.0.0.1:137", 15*time.Second) {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		logger.DebugEnd(Name, "[TIMEOUT][FAILED]")
		return fmt.Errorf("timeout waiting for NetBIOS network interface to bind socket 137")
	}

	s.sState = systeminterface.STARTED
	logger.DebugEnd(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	if s.cmd != nil && s.cmd.Process != nil {
		logger.DebugContinue(Name, "Stopping NetBIOS daemon threads...")

		if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.cmd.Process.Kill()
		}

		_ = s.cmd.Wait()
		logger.DebugAppend(Name, "[CMD Stop]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.DebugAppend(Name, "[STDOUT->LOG STOP]")
	}
	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("netbios background system execution tracking instance is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
