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

type WSDiscoveryStruct struct {
	sState systeminterface.SysStateEnum
	ctx    context.Context
	cancel context.CancelFunc
	engine *engine.Engine

	SkipValidation bool
}

func (_ *WSDiscoveryStruct) Name() string                               { return "wsdiscovery" }
func (_ *WSDiscoveryStruct) Type() systeminterface.SystemTypeEnum       { return systeminterface.Beacon }
func (_ *WSDiscoveryStruct) IsCritical() bool                           { return false }
func (_ *WSDiscoveryStruct) AutoStart() bool                            { return true }
func (w *WSDiscoveryStruct) State(in systeminterface.SysStateEnum) bool { return w.sState == in }

func (w *WSDiscoveryStruct) Setup() {
	logger.Info("WSDiscovery", "Executing service configuration pre-flight routines...")

	templates.PreCompileTemplates()
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.engine = engine.NewEngineState()

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
	w.sState = systeminterface.SETUP
}

func (w *WSDiscoveryStruct) Start() error {
	if w.engine == nil {
		err := fmt.Errorf("setup state was not executed")
		logger.Error("WSDiscovery", "Start process failed fundamentally", err)
		return fmt.Errorf("WSDiscovery service failed to start: %w", err)
	}

	logger.Info("WSDiscovery", "Launching background network engines and dispatcher routines...")
	err := w.engine.Start(
		w.ctx,
		"/config",
	)
	if err != nil {
		logger.Error("WSDiscovery", "Engine failed to initialize completely", err)
		return fmt.Errorf("WSDiscovery engine failed to initialize: %w", err)
	}

	logger.Info("WSDiscovery", "Daemon engine successfully running in background mode.")
	w.sState = systeminterface.STARTED
	return nil
}

func (w *WSDiscoveryStruct) Healthcheck() error {
	if w.ctx != nil && w.ctx.Err() != nil {
		logger.Error("WSDiscovery", "Healthcheck failed: execution context is dropped", w.ctx.Err())
		return fmt.Errorf("WSDiscovery system operational context has dropped: %w", w.ctx.Err())
	}
	logger.Debug("WSDiscovery", "Healthcheck verified successfully. Context is uncorrupted.")
	return nil
}

func (w *WSDiscoveryStruct) Stop() {
	if w.cancel == nil {
		logger.Error("WSDiscovery", "Stop command skipped: cancellation pointer wrapper is unallocated", nil)
		return
	}

	logger.Info("WSDiscovery", "Shutdown execution requested. Safely draining network workers...")
	w.cancel()
	w.engine.Stop()
	logger.Info("WSDiscovery", "Subsystem completely closed down. Multicast groups detached cleanly.")
	w.sState = systeminterface.STOPPED
	return
}
