package logger

import (
	"fmt"
)

func InfoF(subsystem, format string, args ...any) {
	Info(subsystem, fmt.Sprintf(format, args...))
}

func Info(subsystem, message string) {
	prefix := formatPrefix(subsystem, "INFO")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func InfoContinueF(subsystem, format string, args ...any) {
	InfoContinue(subsystem, fmt.Sprintf(format, args...))
}

func InfoContinue(subsystem, message string) {
	prefix := formatPrefix(subsystem, "INFO")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subsystem}
}

func InfoAppendF(subsystem, format string, a ...any) {
	InfoAppend(subsystem, fmt.Sprintf(format, a...))
}

func InfoAppend(subsystem, message string) {
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subsystem}
}

func InfoEndF(subSystem, format string, a ...any) {
	InfoEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func InfoEnd(subSystem, message string) {
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}
