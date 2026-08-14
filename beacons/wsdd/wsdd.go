package wsdd

import (
	"context"
	"fmt"
	"gorogs/beacons"
	"gorogs/beacons/wsdd/engine"
	"gorogs/logger"
)

type WsddBeacon struct {
	config beacons.AppConfig
	ctx    context.Context
	cancel context.CancelFunc
	state  *engine.EngineState
}

func (w *WsddBeacon) Setup(cfg beacons.AppConfig) error {
	w.config = cfg
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.state = engine.NewEngineState()
	logger.Info("wsdd", fmt.Sprintf("Subsystem setup completed for server name: %s", w.config.ServerName))
	return nil
}

func (w *WsddBeacon) Start() error {
	if w.state == nil {
		err := fmt.Errorf("setup state was not executed")
		logger.Error("wsdd", "Start process failed fundamentally", err)
		return fmt.Errorf("wsdd service failed to start: %w", err)
	}

	logger.Info("wsdd", "Launching background network engines and dispatcher routines...")
	err := engine.StartEngine(
		w.ctx,
		w.state,
		w.config.ServerName,
		w.config.ContainerIP.String(),
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
	engine.StopEngine(w.state)
	logger.Info("wsdd", "Subsystem completely closed down. Multicast groups detached cleanly.")
	return nil
}
