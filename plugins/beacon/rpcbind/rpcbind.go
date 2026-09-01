package main

import (
	"fmt"
	"os/exec"
	"syscall"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/system"
)

const (
	Name       string                = "RPCBind"
	Type       system.SystemTypeEnum = system.Beacon
	IsCritical bool                  = false
	AutoStart  bool                  = true
)

var (
	programPath string = "/usr/bin/rpcbind"
	statdPath   string = "/usr/bin/rpc.statd"
)

type Struct struct {
	sState      system.SysStateEnum
	rpcCmd      *exec.Cmd
	statdCmd    *exec.Cmd
	rpcWriter   *config.SubsystemWriter
	statdWriter *config.SubsystemWriter
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

func (s *Struct) Config(cm config.ConfigMap) {}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")
	s.sState = system.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")
	if err := s.startRPCBind(); err != nil {
		return err
	}
	logger.Debug(Name, "[RPCBIND:DONE][RPC.STATD:STARTING]")
	if err := s.startRPCStatd(); err != nil {
		return err
	}
	s.sState = system.STARTED
	logger.Debug(Name, "[RPC.STATD:DONE][DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.Debug(Name, "Stopping RPCBind...")

	s.stopRPCStatd()
	s.stopRPCBind()

	s.sState = system.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.rpcCmd == nil || s.rpcCmd.Process == nil {
		return fmt.Errorf("RPCBind is not initialized")
	}
	return s.rpcCmd.Process.Signal(syscall.Signal(0))
}
