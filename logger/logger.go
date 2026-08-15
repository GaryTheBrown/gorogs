package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
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

func ErrorF(subsystem, format string, err error, args ...any) {
	Error(subsystem, fmt.Sprintf(format, args...), err)
}
func Error(subsystem, message string, err error) {
	prefix := formatPrefix(subsystem, "ERROR")
	logChan <- logMessage{kind: typeStandard, text: prefix + message, subSystem: subsystem}
}

func FatalF(subsystem, format string, err error, args ...any) {

	callerInfo := "Unknown Location"
	if _, file, line, ok := runtime.Caller(1); ok {
		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			shortFile = file[idx+1:]
		}
		callerInfo = fmt.Sprintf("%s:%d", shortFile, line)
	}

	fatalWithCaller(subsystem, fmt.Sprintf(format, args...), err, callerInfo)
}

func Fatal(subsystem, message string, err error) {
	callerInfo := "Unknown Location"
	if _, file, line, ok := runtime.Caller(1); ok {
		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			shortFile = file[idx+1:]
		}
		callerInfo = fmt.Sprintf("%s:%d", shortFile, line)
	}

	fatalWithCaller(subsystem, message, err, callerInfo)
}

func fatalWithCaller(subsystem, message string, err error, callerInfo string) {
	prefix := formatPrefix(subsystem, "FATAL CRASH")

	pid := os.Getpid()
	goVersion := runtime.Version()
	platformArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	heapAllocMB := float64(memStats.HeapAlloc) / 1024 / 1024
	goroutineCount := runtime.NumGoroutine()

	containerContext := "Standard Linux Environment"
	if _, errCheck := os.Stat("/.dockerenv"); errCheck == nil {
		containerContext = "Docker Namespace Container"
	}

	var sb strings.Builder
	sb.WriteString("\n==================================================================\n")

	if err != nil {
		sb.WriteString(fmt.Sprintf(" %s %s\n     |- System Error context: %v\n",
			prefix, message, err))
	} else {
		sb.WriteString(fmt.Sprintf("%s %s\n", prefix, message))
	}

	sb.WriteString(fmt.Sprintf("     |- Failure Origin Location : %s\n", callerInfo))
	sb.WriteString(fmt.Sprintf("     |- Process Identifier  : PID %d\n", pid))
	sb.WriteString(fmt.Sprintf("     |- Active Runtime Scope : %s\n", containerContext))
	sb.WriteString(fmt.Sprintf("     |- Concurrent Routines : %d active loops\n", goroutineCount))
	sb.WriteString(fmt.Sprintf("     |- Allocated Heap RAM  : %.2f MB\n", heapAllocMB))
	sb.WriteString(fmt.Sprintf("     |- Compiler Runtime    : %s\n", goVersion))
	sb.WriteString(fmt.Sprintf("     |- Target Architecture : %s\n", platformArch))
	sb.WriteString("==================================================================\n\n")

	fatalChan <- sb.String()

	<-fatalAck

	os.Exit(1)
}
