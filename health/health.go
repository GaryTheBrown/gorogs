package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"

	"gorogs/beacons"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/shares"
)

// Define the absolute local path for the Unix domain socket file
const socketPath = "/run/gorogs-health.sock"

var (
	TrackedShares  = make(map[string]shares.StorageShare)
	TrackedBeacons = make(map[string]beacons.DiscoveryBeacon)
)

// RunHealthProbeClient triggers natively inside the container namespace via the --check-health flag
func RunHealthProbeClient() {
	// Connect straight to the local Unix socket file system pointer
	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	// The hostname part of the URL ("http://unix") is ignored by our local dialer
	resp, err := client.Get("http://unix/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func StartHealthServer() {
	// Clean up any stale socket files left over from a dirty container crash/restart
	_ = os.Remove(socketPath)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		isHealthy := true
		failureMessage := ""
		currentMode := config.HealthMode

		logger.Debug("HEALTH", fmt.Sprintf("Executing active evaluation loop under strategy level code: %d", currentMode))

		if currentMode == config.LevelDisabled {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
			return
		}

		// 1. Process Core Storage Shares Matrix
		for name, share := range TrackedShares {
			if currentMode == config.LevelNfs && name != "nfs" {
				continue
			}
			if currentMode == config.LevelSamba && name != "samba" {
				continue
			}

			err := share.Healthcheck()
			if err != nil {
				shouldFail := false
				switch currentMode {
				case config.LevelFull, config.LevelShares, config.LevelNfs, config.LevelSamba:
					shouldFail = true
				case config.LevelCritical, config.LevelDefault:
					shouldFail = share.IsCritical()
				}

				if shouldFail {
					isHealthy = false
					failureMessage = fmt.Sprintf("Critical storage share error on component [%s]", name)
					logger.Error("HEALTH", failureMessage, err)
					break
				}
			}
		}

		// 2. Process Auxiliary Network Beacons Matrix
		if isHealthy && currentMode != config.LevelShares && currentMode != config.LevelNfs && currentMode != config.LevelSamba {
			for name, beacon := range TrackedBeacons {
				err := beacon.Healthcheck()
				if err != nil {
					shouldFail := false
					switch currentMode {
					case config.LevelFull:
						shouldFail = true
					case config.LevelCritical:
						shouldFail = beacon.IsCritical()
					}

					if shouldFail {
						isHealthy = false
						failureMessage = fmt.Sprintf("Critical advertisement beacon error on component [%s]", name)
						logger.Error("HEALTH", failureMessage, err)
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

	go func() {
		// Listen natively on the Unix domain socket interface layer
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			logger.Error("HEALTH", "Failed to bind to local Unix socket path", err)
			return
		}

		// Restrict file permission so any user execution context inside the container can safely query it
		_ = os.Chmod(socketPath, 0666)

		if err := http.Serve(listener, nil); err != nil {
			logger.Error("HEALTH", "Internal loop HTTP health server tracking daemon collapsed", err)
		}
	}()
}
