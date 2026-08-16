//go:build debug

package logger

import (
	"fmt"
	"os"
	"strings"
)

const DebugActive = true

var debugRegistry = make(map[string]bool)
var allDebugActive = false

func init() {
	envValue := os.Getenv("DEBUG_LOG")
	if envValue == "" {
		return
	}
	targets := strings.Split(strings.ToLower(envValue), ",")
	for _, target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "all" {
			allDebugActive = true
			break
		}
		debugRegistry[trimmed] = true
	}
}

func DebugF(subsystem, format string, args ...any) {
	Debug(subsystem, fmt.Sprintf(format, args...))
}

func Debug(subsystem, message string) {
	lowerSub := strings.ToLower(subsystem)
	if !allDebugActive && !debugRegistry[lowerSub] {
		return
	}
	prefix := formatPrefix(subsystem, "DEBUG")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func DebugContinueF(subsystem, format string, args ...any) {
	DebugContinue(subsystem, fmt.Sprintf(format, args...))
}

func DebugContinue(subsystem, message string) {
	prefix := formatPrefix(subsystem, "DEBUG")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subsystem}
}

func DebugAppendF(subsystem, format string, a ...any) {
	DebugAppend(subsystem, fmt.Sprintf(format, a...))
}

func DebugAppend(subsystem, message string) {
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subsystem}
}

func DebugEndF(subSystem, format string, a ...any) {
	DebugEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func DebugEnd(subSystem, message string) {
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}

func IsDebugActive(subsystem string) bool {
	if allDebugActive {
		return true
	}
	return debugRegistry[strings.ToLower(subsystem)]
}
