package rpcbind

import (
	"fmt"
	"os/exec"
	"syscall"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "RPCBind"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

var (
	programPath = "/usr/bin/rpcbind"
	statdPath   = "/usr/bin/rpc.statd"
)

type Struct struct {
	sState      systeminterface.SysStateEnum
	rpcCmd      *exec.Cmd
	statdCmd    *exec.Cmd
	rpcWriter   *helpers.SubsystemWriter
	statdWriter *helpers.SubsystemWriter
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Config(cm config.ConfigMap) {
	// NOTHING TO CONFIGURE IN HERE
}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")
	s.sState = systeminterface.SETUP
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
	s.sState = systeminterface.STARTED
	logger.Debug(Name, "[RPC.STATD:DONE][DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.Debug(Name, "Stopping RPCBind...")

	s.stopRPCStatd()
	s.stopRPCBind()

	s.sState = systeminterface.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.rpcCmd == nil || s.rpcCmd.Process == nil {
		return fmt.Errorf("RPCBind is not initialized")
	}
	return s.rpcCmd.Process.Signal(syscall.Signal(0))
}
