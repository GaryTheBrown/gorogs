package systems

import (
	"gorogs/config"
	"gorogs/logger"
)

func ShouldItStart(e SystemNameEnum) bool {
	systemName := systemList[e].Name()
	logger.DebugF(logName, "Is It [%s] Starting, Autostart: [%t] Disabled: [%t] Enabled: [%t]",
		systemName, systemList[e].AutoStart(), config.IsDisabled(systemName), config.IsEnabled(systemName))
	return (systemList[e].AutoStart() && !config.IsDisabled(systemName)) ||
		(!systemList[e].AutoStart() && config.IsEnabled(systemName))
}

func SystemNameList() []string {
	var snList []string
	for _, sys := range systemList {
		snList = append(snList, sys.Name())
	}
	return snList
}
