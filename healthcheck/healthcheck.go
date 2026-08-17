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
	"gorogs/systems/systeminterface"
)

const socketFile = "/run/gorogs-healtsock"
const logName = "healthcheck"

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
	healthMode       Level
	trackedBeacons   map[string]systeminterface.System
	trackedShares    map[string]systeminterface.System
	trackedUtilities map[string]systeminterface.System
	server           *http.Server
	listener         net.Listener
)

func Setup() error {
	logger.DebugContinue(logName, "System Setup...")
	trackedBeacons = make(map[string]systeminterface.System)
	logger.DebugAppend(logName, "[make map]")
	trackedShares = make(map[string]systeminterface.System)
	logger.DebugAppend(logName, "[make map]")
	trackedUtilities = make(map[string]systeminterface.System)
	logger.DebugAppend(logName, "[make map]")

	hEnv := strings.ToLower(config.GetSingleServiceConfigString(logName, "default"))
	logger.DebugAppend(logName, "[get healthmode]")
	switch hEnv {
	case "full":
		healthMode = Full
		logger.DebugAppend(logName, "[set healthmode][FULL]")
	case "critical":
		healthMode = Critical
		logger.DebugAppend(logName, "[set healthmode][CRITICAL]")
	case "shares":
		healthMode = Shares
		logger.DebugAppend(logName, "[set healthmode][SHARES]")
	case "nfs":
		healthMode = Nfs
		logger.DebugAppend(logName, "[set healthmode][NFS]")
	case "samba":
		healthMode = Samba
		logger.DebugAppend(logName, "[set healthmode][SAMBA]")
	case "disabled":
		healthMode = Disabled
		logger.DebugAppend(logName, "[set healthmode][DISABLED]")
	default:
		healthMode = Default
		logger.DebugAppend(logName, "[set healthmode][DEFAULT]")
	}

	logger.DebugEnd(logName, "[DONE]")
	return nil
}

func Start() error {
	logger.DebugContinue(logName, "System Starting...")
	_ = os.Remove(socketFile)
	logger.DebugAppend(logName, "[REMOVE OLD SOCKET]")

	mux := http.NewServeMux()
	mux.HandleFunc("/", handler)
	logger.DebugAppend(logName, "[HANDLER ADDED]")

	server = &http.Server{Handler: mux}

	var err error
	listener, err = net.Listen("unix", socketFile)
	if err != nil {
		logger.DebugEnd(logName, "[SOCKET STARTUP FAILED]")
		return fmt.Errorf("failed to bind to local Unix socket path: %w", err)
	}
	logger.DebugAppend(logName, "[SOCKET LISTEN SUCCESS]")

	_ = os.Chmod(socketFile, 0666)
	logger.DebugAppend(logName, "[PERMISSIONS SET]")

	go socketListner()
	logger.DebugEnd(logName, "[SOCKET STARTED][DONE]")

	return nil
}

func Stop() error {
	logger.DebugContinue(logName, "System Stopping...")
	if server == nil {
		return nil
	}
	logger.DebugAppend(logName, "[SOCKET SHUTDOWN]")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		logger.DebugEnd(logName, "[SOCKET SHUTDOWN FAILED]")
		return err
	}

	_ = os.Remove(socketFile)
	logger.DebugEnd(logName, "[REMOVE SOCKET][DONE]")

	return nil
}

func AddTracker(sys systeminterface.System) bool {
	logger.DebugContinueF(logName, "Adding Tracker for %s...", sys.Name())
	lName := strings.ToLower(sys.Name())
	switch sys.Type() {
	case systeminterface.Beacon:
		trackedBeacons[lName] = sys
		logger.DebugAppend(logName, "[TYPE BEACON]")
	case systeminterface.Share:
		trackedShares[lName] = sys
		logger.DebugAppend(logName, "[TYPE SHARE]")
	case systeminterface.Utility:
		trackedUtilities[lName] = sys
		logger.DebugAppend(logName, "[TYPE UTILITY]")
	default:
		logger.DebugEnd(logName, "[TYPE UNKNOWN]")
		return false
	}
	logger.DebugEnd(logName, "[DONE]")
	return true
}

func socketListner() {
	logger.Debug(logName+".SocketListner", "Serving.. [BLOCKING]")

	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(logName, "Internal loop HTTP health server tracking daemon collapsed", err)
		return
	}

	logger.Debug(logName+".SocketListner", "Closed")
}
