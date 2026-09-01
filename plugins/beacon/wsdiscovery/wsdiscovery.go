package main

import (
	"context"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/plugins/beacon/wsdiscovery/connection"
	"gorogs/plugins/beacon/wsdiscovery/engine"
	"gorogs/plugins/beacon/wsdiscovery/incoming"
	"gorogs/plugins/beacon/wsdiscovery/templates"
	"gorogs/plugins/beacon/wsdiscovery/versions"
	"gorogs/system"
	"time"
)

const (
	Name       = "WSDiscovery"
	Type       = system.Beacon
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
	sState system.SysStateEnum
	ctx    context.Context
	cancel context.CancelFunc
	engine *engine.Engine

	SkipValidation bool
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

func (s *Struct) Config(cm config.ConfigMap) {
	incoming.DisableValidation = cm.Get("disablevalidation", false)
	connection.FastDecodingMode = cm.Get("fastdecode", false)
}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")

	if err := templates.PreCompileTemplates(); err != nil {

	}
	logger.Debug(Name, "[PRECOMPILE TEMPLATES]")
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.engine = engine.NewEngineState()
	logger.Debug(Name, "[SETUP ENGINE]")

	s.sState = system.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")
	err := s.engine.Start(
		s.ctx,
		"/config",
	)
	if err != nil {
		return fmt.Errorf("WSDiscovery engine failed to initialize: %w", err)
	}
	logger.Debug(Name, "[STARTED ENGINE]")

	s.sState = system.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.Debug(Name, "Stopping WSDiscovery...")

	if s.engine != nil {
		done := make(chan struct{})

		go func() {
			s.engine.Stop()
			close(done)
		}()

		logger.Debug(Name, "[CMD Stop Initiated]")

		select {
		case <-done:
			logger.Debug(Name, "[ENGINE INTERNAL FLUSH CLEAN]")
		case <-s.engine.FlushDone:
			logger.Debug(Name, "[BROADCAST BYE COUPLING CONFIRMED]")
		case <-time.After(300 * time.Millisecond):
			logger.Debug(Name, "[TIMEOUT PROTECTION TRIGGERED]")
		}
	}

	if s.cancel != nil {
		s.cancel()
		logger.Debug(Name, "[CONTEXT CANCEL CLEAN]")
	}

	s.sState = system.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.ctx != nil && s.ctx.Err() != nil {
		return fmt.Errorf("WSDiscovery is not initialized")
	}
	return nil
}
