package samba

import (
	"gorogs/logger"
	"gorogs/plugins/share/samba/modes"
	"gorogs/plugins/share/samba/structs"
)

func (s *Struct) setupSystem() {
	modeStr := structs.ModeToString(systemMode)
	logger.DebugF(Name, "[MODE:%s]", modeStr)
	cm := modes.SharedConfigFile(systemMode)
	sm := structs.NewShareMap()
	logger.DebugF(Name, "[SHARES: Count(%d)]", sm.Count())
	switch systemMode {
	case structs.ModeFile:
		s.sys = &modes.ModeFile{
			ConfigMap: cm,
			SharesMap: sm,
		}
	case structs.ModeMixed:
		s.sys = &modes.ModeMixed{
			ConfigMap: cm,
			SharesMap: sm,
		}
	case structs.ModeRegistry:
		s.sys = &modes.ModeRegistry{
			ConfigMap: cm,
			SharesMap: sm,
		}
	default:
		logger.FatalF(Name, "failed to get Sambas system mode. Got Int %d", nil, int(systemMode))
	}

	logger.DebugF(Name, "[MODE %s SETUP]", modeStr)
	if err := s.sys.Setup(); err != nil {
		logger.FatalF(Name, "failed to Setup the Mode [%s]", err, modeStr)
	}
}
