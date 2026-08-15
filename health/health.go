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

const socketPath = "/run/gorogs-health.sock"

var (
	TrackedShares  = make(map[string]shares.StorageShare)
	TrackedBeacons = make(map[string]beacons.DiscoveryBeacon)
)

func RunHealthProbeClient() {

	client := http.Client{
		Transport: &http.Transport{
			DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
				return net.Dial("unix", socketPath)
			},
		},
	}

	resp, err := client.Get("http://unix/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func StartHealthServer() {
	_ = os.Remove(socketPath)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		isHealthy := true
		failureMessage := ""
		currentMode := config.HealthMode

		logger.DebugF("HEALTH", "Executing active evaluation loop under strategy level code: %d", currentMode)

		if currentMode == config.LevelDisabled {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
			return
		}

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
					logger.ErrorF("HEALTH", "Critical storage share error on component [%s]", err, name)
					break
				}
			}
		}

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

	go func() {
		listener, err := net.Listen("unix", socketPath)
		if err != nil {
			logger.Error("HEALTH", "Failed to bind to local Unix socket path", err)
			return
		}

		_ = os.Chmod(socketPath, 0666)

		if err := http.Serve(listener, nil); err != nil {
			logger.Error("HEALTH", "Internal loop HTTP health server tracking daemon collapsed", err)
		}
	}()
}
