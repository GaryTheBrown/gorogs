package logger

import (
	"fmt"
)

func ErrorF(subSystem, format string, err error, args ...any) {
	Error(subSystem, fmt.Sprintf(format, args...), err)
}
func Error(subSystem, message string, err error) {
	prefix := formatPrefix(subSystem, "ERROR")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}
