package logger

import (
	"fmt"
)

func WarnF(subSystem, format string, args ...any) {
	Warn(subSystem, fmt.Sprintf(format, args...))
}

func Warn(subSystem, message string) {
	prefix := formatPrefix(subSystem, "WARN")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func WarnContinueF(subSystem, format string, args ...any) {
	WarnContinue(subSystem, fmt.Sprintf(format, args...))
}

func WarnContinue(subSystem, message string) {
	prefix := formatPrefix(subSystem, "WARN")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subSystem}
}

func WarnAppendF(subSystem, format string, a ...any) {
	WarnAppend(subSystem, fmt.Sprintf(format, a...))
}

func WarnAppend(subSystem, message string) {
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subSystem}
}

func WarnEndF(subSystem, format string, a ...any) {
	WarnEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func WarnEnd(subSystem, message string) {
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}
