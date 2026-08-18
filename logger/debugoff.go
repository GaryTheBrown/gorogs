//go:build !debug

package logger

const DebugActive = false

func DebugF(subSystem, format string, args ...any)         {}
func Debug(subSystem, message string)                      {}
func DebugContinueF(subSystem, format string, args ...any) {}
func DebugContinue(subSystem, message string)              {}
func DebugAppendF(subSystem, format string, a ...any)      {}
func DebugAppend(subSystem, message string)                {}
func DebugEndF(subSystem, format string, a ...any)         {}
func DebugEnd(subSystem, message string)                   {}

func IsDebugActive(subSystem string) bool { return false }
