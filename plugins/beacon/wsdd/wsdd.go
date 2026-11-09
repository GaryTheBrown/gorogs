package main

import (
	"fmt"
	"gorogs/config"
	"gorogs/helpers"
	"gorogs/logger"
	"gorogs/subsystemwriter"
	"gorogs/system"
	"os/exec"
	"syscall"
	"time"
)

const (
	Name       string                = "wsdd"
	Type       system.SystemTypeEnum = system.Beacon
	IsCritical bool                  = false
	AutoStart  bool                  = true
)

var (
	programPath string = "/usr/bin/wsdd2"
)

type Struct struct {
	sState    system.SysStateEnum
	cmd       *exec.Cmd
	logWriter *subsystemwriter.SubsystemWriter
}

func (_ *Struct) Name() string                        { return Name }
func (_ *Struct) Type() system.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                    { return IsCritical }
func (_ *Struct) AutoStart() bool                     { return AutoStart }
func (s *Struct) IsState(in system.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() system.SysStateEnum       { return s.sState }
func (_ *Struct) Dependencies() []string              { return []string{"Samba"} }
func (_ *Struct) OrderAfter() []string                { return nil }
func (_ *Struct) Priority() int                       { return 100 }

var SystemInstance Struct

func init() {
	system.Register(&SystemInstance)
}

func (s *Struct) Config(cm config.ConfigMap) {}

func (s *Struct) Setup() {
	s.sState = system.SETUP
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")
	fqdn := fmt.Sprint(config.Hostname, ".", config.DomainName)

	args := []string{"-4", "-G", config.Workgroup, "-H", fqdn}
	if logger.IsDebugActive(Name) {
		args = append(args, "-L", "-L", "-W", "-W")
	}

	s.cmd = exec.Command(programPath, args...)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	s.logWriter = subsystemwriter.NewSubsystemWriter(Name, nil)
	s.cmd.Stdout = s.logWriter
	s.cmd.Stderr = s.logWriter
	logger.Debug(Name, "[LINK STDOUT->LOG]")

	if err := s.cmd.Start(); err != nil {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to start standalone %s beacon process: %w", Name, err)
	}
	logger.Debug(Name, "[CMD START]")

	if !helpers.WaitForSocket("udp4", "127.0.0.1:3702", 15*time.Second) {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		logger.Debug(Name, "[TIMEOUT][FAILED]")
		return fmt.Errorf("timeout waiting for %s network interface to bind socket 3702", Name)
	}

	s.sState = system.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.DebugF(Name, "Stopping %s...", Name)
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
		return fmt.Errorf("%s is not initialized", Name)
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
