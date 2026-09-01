package healthcheck

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/system"
)

const (
	socketFile string = "/run/gorogs-healtsock"
	logName    string = "healthcheck"
)

type Level int

const (
	Default Level = iota
	Full
	Critical
	Shares
	Nfs
	Samba
	Disabled
)

var (
	healthMode Level
	tracked    map[string]system.System
	server     *http.Server
	listener   net.Listener
)

func Setup() {
	logger.Debug(logName, "System Setup...")
	tracked = make(map[string]system.System)
	logger.Debug(logName, "[make map]")

	hEnv := strings.ToLower(config.Get(logName, "default"))
	logger.Debug(logName, "[get healthmode]")
	switch hEnv {
	case "full":
		healthMode = Full
		logger.Debug(logName, "[set healthmode][FULL]")
	case "critical":
		healthMode = Critical
		logger.Debug(logName, "[set healthmode][CRITICAL]")
	case "shares":
		healthMode = Shares
		logger.Debug(logName, "[set healthmode][SHARES]")
	case "nfs":
		healthMode = Nfs
		logger.Debug(logName, "[set healthmode][NFS]")
	case "samba":
		healthMode = Samba
		logger.Debug(logName, "[set healthmode][SAMBA]")
	case "disabled":
		healthMode = Disabled
		logger.Debug(logName, "[set healthmode][DISABLED]")
	default:
		healthMode = Default
		logger.Debug(logName, "[set healthmode][DEFAULT]")
	}

	logger.AddHealthCheckStopFunction(Stop)
	logger.Debug(logName, "[DONE]")
}

func Start() error {
	logger.Debug(logName, "System Starting...")
	_ = os.Remove(socketFile)
	logger.Debug(logName, "[REMOVE OLD SOCKET]")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	logger.Debug(logName, "[HANDLER ADDED]")

	server = &http.Server{Handler: mux}

	var err error
	listener, err = net.Listen("unix", socketFile)
	if err != nil {
		logger.Debug(logName, "[SOCKET STARTUP FAILED]")
		return fmt.Errorf("failed to bind to local Unix socket path: %w", err)
	}
	logger.Debug(logName, "[SOCKET LISTEN SUCCESS]")

	_ = os.Chmod(socketFile, 0666)
	logger.Debug(logName, "[PERMISSIONS SET]")

	go socketListner()
	logger.Debug(logName, "[SOCKET STARTED][DONE]")

	return nil
}

func Stop() {
	if server == nil {
		return
	}
	logger.Debug(logName, "System Stopping...")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	logger.Debug(logName, "[SOCKET SHUTDOWN]")
	err := server.Shutdown(ctx)
	if err != nil {
		logger.Debug(logName, "[FAILED]")
		return
	}

	_ = os.Remove(socketFile)
	logger.Debug(logName, "[REMOVE SOCKET][DONE]")

}

func AddTracker(sys system.System) {
	logger.DebugF(logName, "Adding Tracker for %s...", sys.Name())
	lName := strings.ToLower(sys.Name())
	tracked[lName] = sys
	logger.Debug(logName, "[DONE]")
}

func socketListner() {
	logger.Debug(logName+".SocketListner", "Serving.. [BLOCKING]")

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(logName, "Internal loop HTTP health server tracking daemon collapsed", err)
		return
	}

	logger.Debug(logName+".SocketListner", "Closed")
}
