package systems

import (
	"gorogs/config"
	"gorogs/logger"
)

func ShouldItStart(e SystemNameEnum) bool {
	systemName := systemList[e].Name()

	autoStart := systemList[e].AutoStart()
	isDisabled := config.IsDisabled(systemName)
	isEnabled := config.IsEnabled(systemName)

	logger.InfoF(logName, "Evaluating service [%s] -> AutoStart: %t, Explicitly Disabled: %t, Explicitly Enabled: %t",
		systemName, autoStart, isDisabled, isEnabled)

	if autoStart {
		return !isDisabled
	}
	return isEnabled
}

func SystemNameList() []string {
	var snList []string
	for _, sys := range systemList {
		snList = append(snList, sys.Name())
	}
	return snList
}
