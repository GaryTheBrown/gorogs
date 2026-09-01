//go:build debug

package logger

import (
	"fmt"
	"os"
	"strings"
)

const (
	DebugActive bool = true
)

var (
	debugRegistry  map[string]bool = make(map[string]bool)
	allDebugActive bool            = false
)

func init() {
	envValue := os.Getenv("DEBUG_LOG")
	if envValue == "" {
		return
	}
	targets := strings.SplitSeq(strings.ToLower(envValue), ",")
	for target := range targets {
		trimmed := strings.TrimSpace(target)
		if trimmed == "all" {
			allDebugActive = true
			break
		}
		debugRegistry[trimmed] = true
	}
}
func debugCheck(subSystem string) bool {
	if allDebugActive {
		return true
	}
	lowerSub := strings.ToLower(subSystem)
	if strings.Contains(subSystem, ".") {
		splitStrs := strings.Split(lowerSub, ".")
		if debugRegistry[splitStrs[0]] {
			return true
		}
	} else if debugRegistry[lowerSub] {
		return true
	}
	return false
}

func DebugF(subSystem, format string, args ...any) {
	Debug(subSystem, fmt.Sprintf(format, args...))
}

func Debug(subSystem, message string) {
	if !debugCheck(subSystem) {
		return
	}
	prefix := formatPrefix(subSystem, "DEBUG")
	logChan <- logMessage{kind: typeStandard, text: prefix + message + "\n"}
}

func DebugContinueF(subSystem, format string, args ...any) {
	DebugContinue(subSystem, fmt.Sprintf(format, args...))
}

func DebugContinue(subSystem, message string) {
	if !debugCheck(subSystem) {
		return
	}
	prefix := formatPrefix(subSystem, "DEBUG")
	logChan <- logMessage{kind: typeStart, text: prefix + message, subSystem: subSystem}
}

func DebugAppendF(subSystem, format string, a ...any) {
	DebugAppend(subSystem, fmt.Sprintf(format, a...))
}

func DebugAppend(subSystem, message string) {
	if !debugCheck(subSystem) {
		return
	}
	logChan <- logMessage{kind: typeAppend, text: message, subSystem: subSystem}
}

func DebugEndF(subSystem, format string, a ...any) {
	DebugEnd(subSystem, fmt.Sprintf(format, a...)+"\n")
}
func DebugEnd(subSystem, message string) {
	if !debugCheck(subSystem) {
		return
	}
	logChan <- logMessage{kind: typeEnd, text: message + "\n", subSystem: subSystem}
}

func IsDebugActive(subSystem string) bool {
	if allDebugActive {
		return true
	}
	return debugRegistry[strings.ToLower(subSystem)]
}
