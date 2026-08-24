package logstruct

import "gorogs/logger"

func (l *LogSystem) InfoF(format string, args ...any) {
	logger.InfoF(l.SystemName, format, args...)
}

func (l *LogSystem) Info(message string) {
	logger.Info(l.SystemName, message)
}

func (l *LogSystem) InfoContinueF(format string, args ...any) {
	logger.InfoContinueF(l.SystemName, format, args...)
}

func (l *LogSystem) InfoContinue(message string) {
	logger.InfoContinue(l.SystemName, message)
}
func (l *LogSystem) InfoAppendF(format string, a ...any) {
	logger.InfoAppendF(l.SystemName, format, a...)
}

func (l *LogSystem) InfoAppend(message string) {
	logger.InfoAppend(l.SystemName, message)
}

func (l *LogSystem) InfoEndF(format string, a ...any) {
	logger.InfoEndF(l.SystemName, format, a...)
}
func (l *LogSystem) InfoEnd(message string) {
	logger.InfoEnd(l.SystemName, message)
}
