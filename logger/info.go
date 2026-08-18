package logger

import (
	"fmt"
)

func InfoF(subSystem, format string, args ...any) {
	Info(subSystem, fmt.Sprintf(format, args...))
}

func Info(subSystem, message string) {
	prefix := formatPrefix(subSystem, "INFO")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func InfoContinueF(subSystem, format string, args ...any) {
	InfoContinue(subSystem, fmt.Sprintf(format, args...))
}

func InfoContinue(subSystem, message string) {
	prefix := formatPrefix(subSystem, "INFO")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subSystem}
}

func InfoAppendF(subSystem, format string, a ...any) {
	InfoAppend(subSystem, fmt.Sprintf(format, a...))
}

func InfoAppend(subSystem, message string) {
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subSystem}
}

func InfoEndF(subSystem, format string, a ...any) {
	InfoEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func InfoEnd(subSystem, message string) {
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}
