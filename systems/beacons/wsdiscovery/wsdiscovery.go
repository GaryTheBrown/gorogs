package wsdiscovery

import (
	"context"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/engine"
	"gorogs/systems/beacons/wsdiscovery/incoming"
	"gorogs/systems/beacons/wsdiscovery/templates"
	"gorogs/systems/systeminterface"
)

type Struct struct {
	sState systeminterface.SysStateEnum
	ctx    context.Context
	cancel context.CancelFunc
	engine *engine.Engine

	SkipValidation bool
}

func (_ *Struct) Name() string                                 { return "wsdiscovery" }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return systeminterface.Beacon }
func (_ *Struct) IsCritical() bool                             { return false }
func (_ *Struct) AutoStart() bool                              { return true }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	logger.Info("WSDiscovery", "Executing service configuration pre-flight routines...")

	templates.PreCompileTemplates()
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.engine = engine.NewEngineState()

	//Configs For This this will eventually be something we get from the cfg when we
	// switch to a more dynamic way of passing configs in and out.
	incoming.SkipValidation = false //only works if fastdecodingmode is false
	connection.FastDecodingMode = false

	if incoming.SkipValidation {
		logger.Info("WSDiscovery", "High-speed tokenless XML decoding optimization shunt is ACTIVE.")
	} else {
		logger.Info("WSDiscovery", "Standard full-document recursive namespace token validation scan is ACTIVE.")
	}
	logger.InfoF("WSDiscovery", "Subsystem setup completed for server name: %s", config.Hostname)
	s.sState = systeminterface.SETUP
}

func (s *Struct) Start() error {
	if s.engine == nil {
		err := fmt.Errorf("setup state was not executed")
		logger.Error("WSDiscovery", "Start process failed fundamentally", err)
		return fmt.Errorf("WSDiscovery service failed to start: %w", err)
	}

	logger.Info("WSDiscovery", "Launching background network engines and dispatcher routines...")
	err := s.engine.Start(
		s.ctx,
		"/config",
	)
	if err != nil {
		logger.Error("WSDiscovery", "Engine failed to initialize completely", err)
		return fmt.Errorf("WSDiscovery engine failed to initialize: %w", err)
	}

	logger.Info("WSDiscovery", "Daemon engine successfully running in background mode.")
	s.sState = systeminterface.STARTED
	return nil
}

func (s *Struct) Stop() {
	if s.cancel == nil {
		logger.Error("WSDiscovery", "Stop command skipped: cancellation pointer wrapper is unallocated", nil)
		return
	}

	logger.Info("WSDiscovery", "Shutdown execution requested. Safely draining network workers...")
	s.cancel()
	s.engine.Stop()
	logger.Info("WSDiscovery", "Subsystem completely closed down. Multicast groups detached cleanly.")
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.ctx != nil && s.ctx.Err() != nil {
		logger.Error("WSDiscovery", "Healthcheck failed: execution context is dropped", s.ctx.Err())
		return fmt.Errorf("WSDiscovery system operational context has dropped: %w", s.ctx.Err())
	}
	logger.Debug("WSDiscovery", "Healthcheck verified successfully. Context is uncorrupted.")
	return nil
}
