package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

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
		fmt.Fprintf(&sb, " %s %s\n     |- System Error context: %v\n", prefix, message, err)
	} else {
		fmt.Fprintf(&sb, "%s %s\n", prefix, message)
	}

	fmt.Fprintf(&sb, "     |- Failure Origin Location : %s\n", callerInfo)
	fmt.Fprintf(&sb, "     |- Process Identifier  : PID %d\n", pid)
	fmt.Fprintf(&sb, "     |- Active Runtime Scope : %s\n", containerContext)
	fmt.Fprintf(&sb, "     |- Concurrent Routines : %d active loops\n", goroutineCount)
	fmt.Fprintf(&sb, "     |- Allocated Heap RAM  : %.2f MB\n", heapAllocMB)
	fmt.Fprintf(&sb, "     |- Compiler Runtime    : %s\n", goVersion)
	fmt.Fprintf(&sb, "     |- Target Architecture : %s\n", platformArch)
	sb.WriteString("==================================================================\n\n")

	fatalChan <- sb.String()

	<-fatalAck

	if SystemsStopFunction != nil {
		SystemsStopFunction()
	}
	if HealthCheckStopFunction != nil {
		HealthCheckStopFunction()
	}
	os.Exit(1)
}
