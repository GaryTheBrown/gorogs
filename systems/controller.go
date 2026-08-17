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
}
func Start() error {
	for _, sys := range systemList {
		if sys.State(systeminterface.SETUP) {
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
func Stop() error {
	for _, sys := range slices.Backward(systemList) {
		if sys.State(systeminterface.STARTED) {
			logger.InfoContinueF(sys.Name(), "Stopping [%s] system...", sys.Name())
			sys.Stop()
			logger.InfoEnd(sys.Name(), "[DONE]")
		}

	}
	return nil
}
