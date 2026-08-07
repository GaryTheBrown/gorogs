package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

var debugRegistry = make(map[string]bool)
var allDebugActive = false

func RegisterDebugTargets(envValue string) {
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

func Debug(subsystem, message string) {
	lowerSub := strings.ToLower(subsystem)
	if !allDebugActive && !debugRegistry[lowerSub] {
		return
	}
	timestamp := time.Now().Format("02/01/2006 15:04:05")
	fmt.Printf("[%s] %s[DEBUG]%s [%s] %s\n",
		timestamp, colorCyan, colorReset, strings.ToUpper(subsystem), message)
}

func Info(subsystem, message string) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")
	fmt.Printf("[%s] %s[INFO]%s [%s] %s\n",
		timestamp, colorGreen, colorReset, strings.ToUpper(subsystem), message)
}

func Error(subsystem, message string, err error) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s[ERROR]%s [%s] %s: %v\n",
			timestamp, colorYellow, colorReset, strings.ToUpper(subsystem), message, err)
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] %s[ERROR]%s [%s] %s\n",
		timestamp, colorYellow, colorReset, strings.ToUpper(subsystem), message)
}

func Fatal(subsystem, message string, err error) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")
	pid := os.Getpid()
	goVersion := runtime.Version()
	platformArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	fmt.Fprintf(os.Stderr, "\n==================================================================\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s[FATAL CRASH]%s [%s] %s\n     |- System Error context: %v\n",
			timestamp, colorRed, colorReset, strings.ToUpper(subsystem), message, err)
	} else {
		fmt.Fprintf(os.Stderr, "[%s] %s[FATAL CRASH]%s [%s] %s\n",
			timestamp, colorRed, colorReset, strings.ToUpper(subsystem), message)
	}
	fmt.Fprintf(os.Stderr, "     |- Process Identifier  : PID %d\n", pid)
	fmt.Fprintf(os.Stderr, "     |- Compiler Runtime    : %s\n", goVersion)
	fmt.Fprintf(os.Stderr, "     |- Target Architecture : %s\n", platformArch)
	fmt.Fprintf(os.Stderr, "==================================================================\n\n")

	os.Exit(1)
}

// IsDebugActive allows submodules to verify if their specific flag is toggled on inside the registry
func IsDebugActive(subsystem string) bool {
	if allDebugActive {
		return true
	}
	return debugRegistry[strings.ToLower(subsystem)]
}
