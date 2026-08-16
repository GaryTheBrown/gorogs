package health

import (
	"net"
	"net/http"
	"os"
	"strings"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems"
)

const socketFile = "/run/gorogs-health.sock"
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

type CheckStruct struct {
	healthMode       Level
	trackedBeacons   map[string]systems.System
	trackedShares    map[string]systems.System
	trackedUtilities map[string]systems.System
}

func (h *CheckStruct) Setup() error {
	logger.DebugContinue(logName, "System Setup...")
	h.trackedBeacons = make(map[string]systems.System)
	logger.DebugAppend(logName, "[make map]")
	h.trackedShares = make(map[string]systems.System)
	logger.DebugAppend(logName, "[make map]")
	h.trackedUtilities = make(map[string]systems.System)
	logger.DebugAppend(logName, "[make map]")

	hEnv := strings.ToLower(config.GetSingleServiceConfigString(logName, "default"))
	logger.DebugAppend(logName, "[get healthmode]")
	switch hEnv {
	case "full":
		h.healthMode = Full
		logger.DebugAppend(logName, "[set healthmode][FULL]")
	case "critical":
		h.healthMode = Critical
		logger.DebugAppend(logName, "[set healthmode][CRITICAL]")
	case "shares":
		h.healthMode = Shares
		logger.DebugAppend(logName, "[set healthmode][SHARES]")
	case "nfs":
		h.healthMode = Nfs
		logger.DebugAppend(logName, "[set healthmode][NFS]")
	case "samba":
		h.healthMode = Samba
		logger.DebugAppend(logName, "[set healthmode][SAMBA]")
	case "disabled":
		h.healthMode = Disabled
		logger.DebugAppend(logName, "[set healthmode][DISABLED]")
	default:
		h.healthMode = Default
		logger.DebugAppend(logName, "[set healthmode][DEFAULT]")
	}

	logger.DebugEnd(logName, "[DONE]")
	return nil
}

func (h *CheckStruct) Start() error {
	logger.DebugContinue(logName, "System Starting...")
	_ = os.Remove(socketFile)
	logger.DebugAppend(logName, "[CLEAN OLD SOCKET]")

	http.HandleFunc("/", h.handler)
	logger.DebugAppend(logName, "[HANDLER ADDED]")

	go h.socketListner()
	logger.DebugEnd(logName, "[SOCKET STARTED][DONE]")
	return nil
}

func (h *CheckStruct) Stop() error { return nil }

func (h *CheckStruct) AddTracker(sys systems.System) bool {
	logger.DebugContinueF(logName, "Adding Tracker for %s...", sys.Name())
	switch sys.Type() {
	case systems.Beacon:
		h.trackedBeacons[sys.Name()] = sys
		logger.DebugAppend(logName, "[TYPE BEACON]")
	case systems.Share:
		h.trackedShares[sys.Name()] = sys
		logger.DebugAppend(logName, "[TYPE SHARE]")
	case systems.Utility:
		h.trackedUtilities[sys.Name()] = sys
		logger.DebugAppend(logName, "[TYPE UTILITY]")
	default:
		logger.DebugEnd(logName, "[TYPE UNKNOWN]")
		return false
	}
	logger.DebugEnd(logName, "[Done]")
	return true
}

func (h *CheckStruct) socketListner() {
	logger.Debug(logName+".SocketListner", "Started")
	listener, err := net.Listen("unix", socketFile)
	if err != nil {
		logger.Error(logName, "Failed to bind to local Unix socket path", err)
		return
	}
	logger.Debug(logName+".SocketListner", "bound to local Unix socket path")

	_ = os.Chmod(socketFile, 0666)
	logger.Debug(logName+".SocketListner", "permissions set on local unix socket path")

	logger.Debug(logName+".SocketListner", "Serving.. [BLOCKING]")
	if err := http.Serve(listener, nil); err != nil {
		logger.Error(logName, "Internal loop HTTP health server tracking daemon collapsed", err)
		return
	}
	logger.Debug(logName+".SocketListner", "Closed")
}
