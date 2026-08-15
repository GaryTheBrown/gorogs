//go:build !debug

package logger

func RegisterDebugTargets(envValue string) {}

func DebugF(subsystem, format string, args ...interface{}) {}

func Debug(subsystem, message string) {}

func IsDebugActive(subsystem string) bool { return false }
