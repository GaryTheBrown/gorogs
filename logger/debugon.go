//go:build debug

package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
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
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	prefix := "[DEBUG]"
	if EnableColors {
		prefix = fmt.Sprintf("%s[DEBUG]%s", colorCyan, colorReset)
	}

	fmt.Printf("[%s] %s [%s] %s\n", timestamp, prefix, strings.ToUpper(subsystem), message)
}

func IsDebugActive(subsystem string) bool {
	if allDebugActive {
		return true
	}
	return debugRegistry[strings.ToLower(subsystem)]
}
