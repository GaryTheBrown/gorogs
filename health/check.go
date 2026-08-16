package health

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems"
)

const socketPath = "/run/gorogs-health.sock"

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

func (h *CheckStruct) AddTracker(sys systems.System) bool {
	switch sys.Type() {
	case systems.Beacon:
		h.trackedBeacons[sys.Name()] = sys
	case systems.Share:
		h.trackedShares[sys.Name()] = sys
	case systems.Utility:
		h.trackedUtilities[sys.Name()] = sys
	default:
		return false
	}
	return true
}

func (h *CheckStruct) Setup() error {
	logger.Info("HealthCheck", "Health Checker System Setup...")
	h.trackedBeacons = make(map[string]systems.System)
	h.trackedShares = make(map[string]systems.System)
	h.trackedUtilities = make(map[string]systems.System)

	hEnv := strings.ToLower(config.GetSingleServiceConfigString("healthcheck", "default"))

	switch hEnv {
	case "full":
		h.healthMode = Full
	case "critical":
		h.healthMode = Critical
	case "shares":
		h.healthMode = Shares
	case "nfs":
		h.healthMode = Nfs
	case "samba":
		h.healthMode = Samba
	case "disabled":
		h.healthMode = Disabled
	default:
		h.healthMode = Default
	}

	return nil
}
func (h *CheckStruct) Stop() error { return nil }

func (h *CheckStruct) Start() error {
	_ = os.Remove(socketPath)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		isHealthy := true
		failureMessage := ""

		logger.DebugF("HEALTH", "Executing active evaluation loop under strategy level code: %d", h.healthMode)

		if h.healthMode == Disabled {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
			return
		}

		for name, share := range h.trackedShares {
			if h.healthMode == Nfs && name != "nfs" {
				continue
			}
			if h.healthMode == Samba && name != "samba" {
				continue
			}

			err := share.Healthcheck()
			if err != nil {
				shouldFail := false
				switch h.healthMode {
				case Full, Shares, Nfs, Samba:
					shouldFail = true
				case Critical, Default:
					shouldFail = share.IsCritical()
				}

				if shouldFail {
					isHealthy = false
					logger.ErrorF("HEALTH", "Critical storage share error on component [%s]", err, name)
					break
				}
			}
		}

		if isHealthy && h.healthMode != Shares && h.healthMode != Nfs && h.healthMode != Samba {
			for name, beacon := range h.trackedBeacons {
				err := beacon.Healthcheck()
				if err != nil {
					shouldFail := false
					switch h.healthMode {
					case Full:
						shouldFail = true
					case Critical:
						shouldFail = beacon.IsCritical()
					}

					if shouldFail {
						isHealthy = false
						logger.ErrorF("HEALTH", "Critical advertisement beacon error on component [%s]", err, name)
						break
					}
				}
			}
		}

		if !isHealthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "FAIL: %s\n", failureMessage)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	go socketListner()
	return nil
}

func socketListner() {
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		logger.Error("HEALTH", "Failed to bind to local Unix socket path", err)
		return
	}

	_ = os.Chmod(socketPath, 0666)

	if err := http.Serve(listener, nil); err != nil {
		logger.Error("HEALTH", "Internal loop HTTP health server tracking daemon collapsed", err)
	}
}
