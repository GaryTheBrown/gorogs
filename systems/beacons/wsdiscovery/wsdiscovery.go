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
	"gorogs/systems/beacons/wsdiscovery/versions"
	"gorogs/systems/systeminterface"
	"time"
)

const (
	Name       = "WSDiscovery"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

func init() {
	connection.Name = Name
	engine.Name = Name
	incoming.Name = Name
	versions.Name = Name
	templates.Name = Name
}

type Struct struct {
	sState systeminterface.SysStateEnum
	ctx    context.Context
	cancel context.CancelFunc
	engine *engine.Engine

	SkipValidation bool
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Config() {
	cm := config.GetServiceConfig(Name)
	incoming.DisableValidation = cm.Get("disablevalidation", false)
	connection.FastDecodingMode = cm.Get("fastdecode", false)
}

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	templates.PreCompileTemplates()
	logger.DebugAppend(Name, "[PRECOMPILE TEMPLATES]")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.engine = engine.NewEngineState()
	logger.DebugAppend(Name, "[SETUP ENGINE]")

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")
	err := s.engine.Start(
		s.ctx,
		"/config",
	)
	if err != nil {
		return fmt.Errorf("WSDiscovery engine failed to initialize: %w", err)
	}
	logger.DebugAppend(Name, "[STARTED ENGINE]")

	s.sState = systeminterface.STARTED
	logger.DebugEnd(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.DebugContinue(Name, "Stopping WSDiscovery daemon threads...")

	if s.engine != nil {
		done := make(chan struct{})

		go func() {
			s.engine.Stop()
			close(done)
		}()

		logger.DebugAppend(Name, "[CMD Stop Initiated]")

		select {
		case <-done:
			logger.DebugAppend(Name, "[ENGINE INTERNAL FLUSH CLEAN]")
		case <-s.engine.FlushDone:
			logger.DebugAppend(Name, "[BROADCAST BYE COUPLING CONFIRMED]")
		case <-time.After(300 * time.Millisecond):
			logger.DebugAppend(Name, "[TIMEOUT PROTECTION TRIGGERED]")
		}
	}

	if s.cancel != nil {
		s.cancel()
		logger.DebugAppend(Name, "[CONTEXT CANCEL CLEAN]")
	}

	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.ctx != nil && s.ctx.Err() != nil {
		return fmt.Errorf("WSDiscovery is not initialized")
	}
	return nil
}
