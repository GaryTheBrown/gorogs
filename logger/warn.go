package logger

import (
	"fmt"
)

func WarnF(subsystem, format string, args ...any) {
	Warn(subsystem, fmt.Sprintf(format, args...))
}

func Warn(subsystem, message string) {
	prefix := formatPrefix(subsystem, "WARN")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func WarnContinueF(subsystem, format string, args ...any) {
	WarnContinue(subsystem, fmt.Sprintf(format, args...))
}

func WarnContinue(subsystem, message string) {
	prefix := formatPrefix(subsystem, "WARN")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subsystem}
}

func WarnAppendF(subsystem, format string, a ...any) {
	WarnAppend(subsystem, fmt.Sprintf(format, a...))
}

func WarnAppend(subsystem, message string) {
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subsystem}
}

func WarnEndF(subSystem, format string, a ...any) {
	WarnEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func WarnEnd(subSystem, message string) {
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}
