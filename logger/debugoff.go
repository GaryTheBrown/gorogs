//go:build !debug

package logger

const DebugActive = false

func DebugF(subsystem, format string, args ...any)         {}
func Debug(subsystem, message string)                      {}
func DebugContinueF(subsystem, format string, args ...any) {}
func DebugContinue(subsystem, message string)              {}
func DebugAppendF(subsystem, format string, a ...any)      {}
func DebugAppend(subsystem, message string)                {}
func DebugEndF(subSystem, format string, a ...any)         {}
func DebugEnd(subSystem, message string)                   {}

func IsDebugActive(subsystem string) bool { return false }
