package wsdd

import (
	"context"
	"fmt"
	"gorogs/beacons"
	"gorogs/beacons/wsdd/engine"
	"gorogs/beacons/wsdd/incoming"
	"gorogs/beacons/wsdd/templates"
	"gorogs/logger"
)

type WsddBeacon struct {
	config beacons.AppConfig
	ctx    context.Context
	cancel context.CancelFunc
	engine *engine.Engine

	//Configs For This this will eventually be something we get from the cfg when we
	// switch to a more dynamic way of passing configs in and out.
	FastDecodingMode bool
}

func (w *WsddBeacon) Setup(cfg beacons.AppConfig) error {
	logger.Info("wsdd", "Executing service configuration pre-flight routines...")

	w.config = cfg
	templates.PreCompileTemplates(cfg)
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.engine = engine.NewEngineState()
	w.FastDecodingMode = true
	incoming.EnableFastDecoding = w.FastDecodingMode

	if incoming.EnableFastDecoding {
		logger.Info("WSDD", "High-speed tokenless XML decoding optimization shunt is ACTIVE.")
	} else {
		logger.Info("WSDD", "Standard full-document recursive namespace token validation scan is ACTIVE.")
	}
	logger.Info("wsdd", fmt.Sprintf("Subsystem setup completed for server name: %s", w.config.ServerName))
	return nil
}

func (w *WsddBeacon) Start() error {
	if w.engine == nil {
		err := fmt.Errorf("setup state was not executed")
		logger.Error("wsdd", "Start process failed fundamentally", err)
		return fmt.Errorf("wsdd service failed to start: %w", err)
	}

	logger.Info("wsdd", "Launching background network engines and dispatcher routines...")
	err := w.engine.Start(
		w.ctx,
		w.config,
		"/config",
	)
	if err != nil {
		logger.Error("wsdd", "Engine failed to initialize completely", err)
		return fmt.Errorf("wsdd engine failed to initialize: %w", err)
	}

	logger.Info("wsdd", "Daemon engine successfully running in background mode.")
	return nil
}

func (w *WsddBeacon) Healthcheck() error {
	if w.ctx != nil && w.ctx.Err() != nil {
		logger.Error("wsdd", "Healthcheck failed: execution context is dropped", w.ctx.Err())
		return fmt.Errorf("wsdd system operational context has dropped: %w", w.ctx.Err())
	}
	logger.Debug("wsdd", "Healthcheck verified successfully. Context is uncorrupted.")
	return nil
}

func (w *WsddBeacon) IsCritical() bool { return false }

func (w *WsddBeacon) Stop() error {
	if w.cancel == nil {
		logger.Error("wsdd", "Stop command skipped: cancellation pointer wrapper is unallocated", nil)
		return nil
	}

	logger.Info("wsdd", "Shutdown execution requested. Safely draining network workers...")
	w.cancel()
	w.engine.Stop()
	logger.Info("wsdd", "Subsystem completely closed down. Multicast groups detached cleanly.")
	return nil
}
