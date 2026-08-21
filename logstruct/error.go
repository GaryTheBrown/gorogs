package logstruct

import (
	"gorogs/logger"
)

func (l *LogSystem) ErrorF(format string, err error, args ...any) {
	logger.ErrorF(l.SystemName, format, err, args...)
}
func (l *LogSystem) Error(message string, err error) {
	logger.Error(l.SystemName, message, err)
}
