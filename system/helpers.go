package system

import (
	"gorogs/config"
	"gorogs/system/systeminterface"
)

func ShouldItStart(sys systeminterface.System) bool {
	if sys == nil {
		return false
	}

	systemName := sys.Name()

	if sys.AutoStart() {
		return !config.IsDisabled(systemName)
	}
	return config.IsEnabled(systemName)
}
