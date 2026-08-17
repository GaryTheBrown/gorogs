package systems

import (
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/systems/systeminterface"
	"slices"
)

const logName = "systems"

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
				//Problem Starting system
			}
			if !healthcheck.AddTracker(sys) {
				//problem with adding to tracker
				//if not a critical system maybe allow it to skip adding it here
			}
		}
	}
	return nil
}
func Stop() {
	for _, sys := range slices.Backward(systemList) {
		if sys.IsState(systeminterface.STARTED) {
			logger.InfoContinueF(sys.Name(), "Stopping: %s...", sys.Name())
			sys.Stop()
			if sys.IsState(systeminterface.STOPPED) {
				logger.InfoEnd(sys.Name(), "[DONE]")
			} else {
				logger.InfoEnd(sys.Name(), "[FAILED]")
			}
		}
	}
}
