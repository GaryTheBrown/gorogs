package logstruct

import (
	"gorogs/logger"
)

func (l *LogSystem) FatalF(format string, err error, args ...any) {
	logger.FatalF(l.SystemName, format, err, args...)
}
func (l *LogSystem) Fatal(message string, err error) {
	logger.Fatal(l.SystemName, message, err)
}
