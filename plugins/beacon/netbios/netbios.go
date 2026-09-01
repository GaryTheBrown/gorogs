package main

import (
	"fmt"
	"gorogs/config"
	"gorogs/helpers"
	"gorogs/logger"
	"gorogs/system"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	Name       = "NetBIOS"
	Type       = system.Beacon
	IsCritical = false
	AutoStart  = false
)

var (
	programPath      = "/usr/bin/nmbd"
	masterConfigPath = "/etc/nmbd.conf"
)

type Struct struct {
	sState    system.SysStateEnum
	cmd       *exec.Cmd
	logWriter *config.SubsystemWriter
}

func (_ *Struct) Name() string                        { return Name }
func (_ *Struct) Type() system.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                    { return IsCritical }
func (_ *Struct) AutoStart() bool                     { return AutoStart }
func (s *Struct) IsState(in system.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() system.SysStateEnum       { return s.sState }
func (_ *Struct) Dependencies() []string              { return nil }
func (_ *Struct) OrderAfter() []string                { return nil }
func (_ *Struct) Priority() int                       { return 100 }

var SystemInstance Struct

func init() {
	system.Register(&SystemInstance)
}

func (s *Struct) Config(cm config.ConfigMap) {
}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")

	if err := s.writeMasterNmbdConfig(); err != nil {
		logger.FatalF(Name, "failed to write %s config file", err, Name)
	}
	logger.Debug(Name, "[write config]")

	s.sState = system.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) writeMasterNmbdConfig() error {

	masterContent := fmt.Sprintf(`[global]
	netbios name = %s
	workgroup = %s
	server string = Network Discovery Beacon
	log file = /dev/null
	local master = no
	preferred master = no
	domain master = no
`, config.Hostname, config.Workgroup)
	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")

	s.cmd = exec.Command(programPath, "--foreground", "--no-process-group", "--debug-stdout", "-s", masterConfigPath)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	if logger.IsDebugActive(Name) {
		s.logWriter = config.NewSubsystemWriter(Name, nil)
		s.cmd.Stdout = s.logWriter
		s.cmd.Stderr = s.logWriter
		logger.Debug(Name, "[LINK STDOUT->LOG]")
	}

	if err := s.cmd.Start(); err != nil {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to start standalone nmbd beacon process: %w", err)
	}
	logger.Debug(Name, "[CMD START]")

	if !helpers.WaitForSocket("udp4", "127.0.0.1:137", 15*time.Second) {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		logger.Debug(Name, "[TIMEOUT][FAILED]")
		return fmt.Errorf("timeout waiting for NetBIOS network interface to bind socket 137")
	}

	s.sState = system.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.Debug(Name, "Stopping NetBIOS...")
	if s.cmd != nil && s.cmd.Process != nil {

		if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.cmd.Process.Kill()
		}

		_ = s.cmd.Wait()
		logger.Debug(Name, "[CMD Stop]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.Debug(Name, "[STDOUT->LOG STOP]")
	}
	s.sState = system.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("netbios is not initialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
