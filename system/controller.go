package system

import (
	"context"
	"fmt"
	"gorogs/config"
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/system/systeminterface"
	"slices"
	"strings"
	"sync"
	"time"
)

const logName = "gorogs.controller"

func Config() {
	zeroFreeSpaceStr := "zerofreespace"
	value, found := config.GetExists(zeroFreeSpaceStr, true)

	for _, sys := range systeminterface.SystemList {
		sysConfig := config.GetServiceConfig(sys.Name())
		if found && sys.Type() == systeminterface.Share && !sysConfig.Exists(zeroFreeSpaceStr) {
			sysConfig[zeroFreeSpaceStr] = value
		}
		sys.Config(sysConfig)
	}
}

func Setup() {
	runningStatus := make(map[string]bool)
	var nfsSys systeminterface.System

	for _, sys := range systeminterface.SystemList {
		if strings.EqualFold(sys.Name(), "NFS") {
			nfsSys = sys
		}
		runningStatus[strings.ToLower(sys.Name())] = ShouldItStart(sys)
	}

	for _, sys := range systeminterface.SystemList {
		sysKey := strings.ToLower(sys.Name())
		sysEnabled := runningStatus[sysKey]

		if strings.EqualFold(sys.Name(), "RpcBind") && !sysEnabled && nfsSys != nil && ShouldItStart(nfsSys) {
			sysEnabled = true
			runningStatus[sysKey] = true
		}

		if !sysEnabled {
			logger.DebugF(logName, "Skipping config setup for disabled service: %s", sys.Name())
			continue
		}

		for _, hardDep := range sys.Dependencies() {
			hardDepKey := strings.ToLower(hardDep)
			if !runningStatus[hardDepKey] {
				logger.Fatal(logName, "FATAL DEPENDENCY ERROR",
					fmt.Errorf("cannot start %s because its hard dependency %s is disabled", sys.Name(), hardDep))
			}
		}

		logger.InfoF(logName, "Setting Up: %s", sys.Name())
		sys.Setup()
	}
	logger.AddSystemsStopFunction(Stop)
}

func Start() error {
	for _, sys := range systeminterface.SystemList {
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

	for _, sys := range slices.Backward(systeminterface.SystemList) {
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
