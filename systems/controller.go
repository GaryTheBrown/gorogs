package systems

import (
	"context"
	"gorogs/config"
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/systems/systeminterface"
	"slices"
	"sync"
	"time"
)

const logName = "main"

func Config() {

	for _, sys := range systemList {
		sysConfig := config.GetServiceConfig(sys.Name())
		sys.Config(sysConfig)
	}
}

func Setup() {
	for i, sys := range systemList {
		if !ShouldItStart(SystemNameEnum(i)) ||
			(i == int(RpcBind) && !ShouldItStart(SystemNameEnum(i)) && !ShouldItStart(NFS)) {
			logger.DebugF(logName, "Skipping: %s", sys.Name())
			continue
		}
		logger.InfoF(logName, "Setting Up: %s", sys.Name())
		sys.Setup()
	}
	logger.AddSystemsStopFunction(Stop)
}

func Start() error {
	for _, sys := range systemList {
		if sys.IsState(systeminterface.SETUP) {
			logger.InfoF(logName, "Starting: %s", sys.Name())
			if err := sys.Start(); err != nil {
				logger.Fatal(logName, "SOMETHING FAILED", err)
			}
			healthcheck.AddTracker(sys)
		}
	}
	return nil
}

func Stop() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	for _, sys := range slices.Backward(systemList) {
		if !sys.IsState(systeminterface.STARTED) {
			continue
		}

		wg.Add(1)

		go func(s systeminterface.System) {
			defer wg.Done()

			logger.InfoF(s.Name(), "Stopping: %s", s.Name())
			s.Stop()

			if s.IsState(systeminterface.STOPPED) {
				logger.InfoF(s.Name(), "Stopped: %s[DONE]", s.Name())
			} else {
				logger.InfoF(s.Name(), "Stopped: %s[FAILED]", s.Name())
			}
		}(sys)
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logger.InfoF(logName, "Graceful exit accomplished.")
	case <-shutdownCtx.Done():
		logger.WarnF(logName, "Teardown safety threshold reached before all socket threads cleared. Overriding deadlock to ensure clean exit.")
	}
}
