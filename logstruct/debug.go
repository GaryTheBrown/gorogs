package logstruct

import "gorogs/logger"

func (l *LogSystem) DebugF(format string, args ...any) {
	logger.DebugF(l.SystemName, format, args...)
}

func (l *LogSystem) Debug(message string) {
	logger.Debug(l.SystemName, message)
}

func (l *LogSystem) DebugContinueF(format string, args ...any) {
	logger.DebugContinueF(l.SystemName, format, args...)
}

func (l *LogSystem) DebugContinue(message string) {
	logger.DebugContinue(l.SystemName, message)
}
func (l *LogSystem) DebugAppendF(format string, a ...any) {
	logger.DebugAppendF(l.SystemName, format, a...)
}

func (l *LogSystem) DebugAppend(message string) {
	logger.DebugAppend(l.SystemName, message)
}

func (l *LogSystem) DebugEndF(format string, a ...any) {
	logger.DebugEndF(l.SystemName, format, a...)
}
func (l *LogSystem) DebugEnd(message string) {
	logger.DebugEnd(l.SystemName, message)
}

func IsDebugActive(subSystem string) bool {
	return logger.IsDebugActive(subSystem)
}

func DebugActive() bool {
	return logger.DebugActive
}
