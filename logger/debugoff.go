//go:build !debug 

package logger

const DebugActive = false

func DebugF(subsystem, format string, args ...any) {}

func Debug(subsystem, message string) {}

func IsDebugActive(subsystem string) bool { return false }
