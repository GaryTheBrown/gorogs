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
			continue
		}
		sys.Setup()
	}
	logger.AddSystemsStopFunction(Stop)
}
func Start() error {
	for _, sys := range systemList {
		if sys.IsState(systeminterface.SETUP) {
			if err := sys.Start(); err != nil {
				//Problem Starting system
			}
			if !healthcheck.AddTracker(sys) {
				//problem with adding to tracker
			}
		}
	}
	return nil
}
func Stop() {
	for _, sys := range slices.Backward(systemList) {
		if sys.IsState(systeminterface.STARTED) {
			logger.InfoContinueF(sys.Name(), "Stopping [%s] system...", sys.Name())
			sys.Stop()
			if sys.IsState(systeminterface.STOPPED) {
				logger.InfoEnd(sys.Name(), "[DONE]")
			} else {
				logger.InfoEnd(sys.Name(), "[FAILED]")
			}
		}
	}
}
