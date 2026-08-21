package logstruct

import "gorogs/logger"

func (l *LogSystem) WarnF(format string, args ...any) {
	logger.WarnF(l.SystemName, format, args...)
}

func (l *LogSystem) Warn(message string) {
	logger.Warn(l.SystemName, message)
}

func (l *LogSystem) WarnContinueF(format string, args ...any) {
	logger.WarnContinueF(l.SystemName, format, args...)
}

func (l *LogSystem) WarnContinue(message string) {
	logger.WarnContinue(l.SystemName, message)
}
func (l *LogSystem) WarnAppendF(format string, a ...any) {
	logger.WarnAppendF(l.SystemName, format, a...)
}

func (l *LogSystem) WarnAppend(message string) {
	logger.WarnAppend(l.SystemName, message)
}

func (l *LogSystem) WarnEndF(format string, a ...any) {
	logger.WarnEndF(l.SystemName, format, a...)
}
func (l *LogSystem) WarnEnd(message string) {
	logger.WarnEnd(l.SystemName, message)
}
