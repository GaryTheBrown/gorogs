package system

import (
	"context"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	logName string = "gorogs.system"
)

func Config() {
	zeroFreeSpaceStr := "zerofreespace"
	value, found := config.GetExists(zeroFreeSpaceStr, true)

	for _, sys := range SystemList {
		sysConfig := config.GetServiceConfig(sys.Name())
		if found && sys.Type() == Share && !sysConfig.Exists(zeroFreeSpaceStr) {
			sysConfig[zeroFreeSpaceStr] = value
		}
		sys.Config(sysConfig)
	}
}

func Setup() {
	runningStatus := make(map[string]bool)
	var nfsSys System

	for _, sys := range SystemList {
		if strings.EqualFold(sys.Name(), "NFS") {
			nfsSys = sys
		}
		runningStatus[strings.ToLower(sys.Name())] = ShouldItStart(sys)
	}

	for _, sys := range SystemList {
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
	for _, sys := range SystemList {
		if sys.IsState(SETUP) {
			logger.InfoF(logName, "Starting: %s", sys.Name())
			if err := sys.Start(); err != nil {
				logger.Fatal(logName, "SOMETHING FAILED", err)
			}
		}
	}
	return nil
}

func Stop() {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	for _, sys := range slices.Backward(SystemList) {
		if !sys.IsState(STARTED) {
			continue
		}

		wg.Add(1)

		go func(s System) {
			defer wg.Done()

			logger.InfoF(s.Name(), "Stopping: %s", s.Name())
			s.Stop()

			if s.IsState(STOPPED) {
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
