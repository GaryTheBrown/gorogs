package logger

import (
	"fmt"
)

func ErrorF(subSystem, format string, err error, args ...any) {
	Error(subSystem, fmt.Sprintf(format, args...), err)
}
func Error(subSystem, message string, err error) {
	prefix := formatPrefix(subSystem, "ERROR")
	fullMessage := fmt.Sprintf("%s%s: %v", prefix, message, err.Error())
	logChan <- logMessage{kind: typeStandard, text: fullMessage + "\n"}
}
