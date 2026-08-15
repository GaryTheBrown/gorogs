package logger

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"
)

var EnableColors = true

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

func InfoF(subsystem, format string, args ...interface{}) {
	Info(subsystem, fmt.Sprintf(format, args...))

}
func Info(subsystem, message string) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	prefix := "[INFO]"
	if EnableColors {
		prefix = fmt.Sprintf("%s[INFO]%s", colorGreen, colorReset)
	}

	fmt.Printf("[%s] %s [%s] %s\n", timestamp, prefix, strings.ToUpper(subsystem), message)
}

func ErrorF(subsystem string, format string, err error, args ...interface{}) {
	Error(subsystem, fmt.Sprintf(format, args...), err)
}
func Error(subsystem string, message string, err error) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")

	prefix := "[ERROR]"
	if EnableColors {
		prefix = fmt.Sprintf("%s[ERROR]%s", colorYellow, colorReset)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s [%s] %s: %v\n",
			timestamp, prefix, strings.ToUpper(subsystem), message, err)
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] %s [%s] %s\n", timestamp, prefix, strings.ToUpper(subsystem), message)
}
func FatalF(subsystem string, format string, err error, args ...interface{}) {
	Fatal(subsystem, fmt.Sprintf(format, args...), err)
}
func Fatal(subsystem string, message string, err error) {
	timestamp := time.Now().Format("02/01/2006 15:04:05")
	pid := os.Getpid()
	goVersion := runtime.Version()
	platformArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	callerInfo := "Unknown Location"
	if _, file, line, ok := runtime.Caller(1); ok {
		shortFile := file
		if idx := strings.LastIndex(file, "/"); idx >= 0 {
			shortFile = file[idx+1:]
		}
		callerInfo = fmt.Sprintf("%s:%d", shortFile, line)
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	heapAllocMB := float64(memStats.HeapAlloc) / 1024 / 1024
	goroutineCount := runtime.NumGoroutine()

	containerContext := "Standard Linux Environment"
	if _, errCheck := os.Stat("/.dockerenv"); errCheck == nil {
		containerContext = "Docker Namespace Container"
	}

	prefix := "[FATAL CRASH]"
	if EnableColors {
		prefix = fmt.Sprintf("%s[FATAL CRASH]%s", colorRed, colorReset)
	}

	fmt.Fprintf(os.Stderr, "\n==================================================================\n")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[%s] %s [%s] %s\n     |- System Error context: %v\n",
			timestamp, prefix, strings.ToUpper(subsystem), message, err)
	} else {
		fmt.Fprintf(os.Stderr, "[%s] %s [%s] %s\n", timestamp, prefix, strings.ToUpper(subsystem), message)
	}
	fmt.Fprintf(os.Stderr, "     |- Failure Origin Location : %s\n", callerInfo)
	fmt.Fprintf(os.Stderr, "     |- Process Identifier  : PID %d\n", pid)
	fmt.Fprintf(os.Stderr, "     |- Active Runtime Scope : %s\n", containerContext)
	fmt.Fprintf(os.Stderr, "     |- Concurrent Routines : %d active loops\n", goroutineCount)
	fmt.Fprintf(os.Stderr, "     |- Allocated Heap RAM  : %.2f MB\n", heapAllocMB)
	fmt.Fprintf(os.Stderr, "     |- Compiler Runtime    : %s\n", goVersion)
	fmt.Fprintf(os.Stderr, "     |- Target Architecture : %s\n", platformArch)
	fmt.Fprintf(os.Stderr, "==================================================================\n\n")

	os.Exit(1)
}
