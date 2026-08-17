package logger

import (
	"fmt"
)

func ErrorF(subsystem, format string, err error, args ...any) {
	Error(subsystem, fmt.Sprintf(format, args...), err)
}
func Error(subsystem, message string, err error) {
	prefix := formatPrefix(subsystem, "ERROR")
	logChan <- logMessage{kind: typeStandard, text: prefix + message, subSystem: subsystem}
}
