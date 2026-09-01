package system

import (
	"gorogs/config"
)

func ShouldItStart(sys System) bool {
	if sys == nil {
		return false
	}

	systemName := sys.Name()

	if sys.AutoStart() {
		return !config.IsDisabled(systemName)
	}
	return config.IsEnabled(systemName)
}
