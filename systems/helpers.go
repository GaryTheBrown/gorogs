package systems

import (
	"gorogs/config"
)

func ShouldItStart(e SystemNameEnum) bool {
	systemName := systemList[e].Name()

	autoStart := systemList[e].AutoStart()
	isDisabled := config.IsDisabled(systemName)
	isEnabled := config.IsEnabled(systemName)

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
