package health

import (
	"fmt"
	"net/http"
	"os"

	"gorogs/beacons"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/shares"
)

// Updated maps to reference the full structural interfaces natively
var (
	TrackedShares  = make(map[string]shares.StorageShare)
	TrackedBeacons = make(map[string]beacons.DiscoveryBeacon)
)

func RunHealthProbeClient() {
	resp, err := http.Get("http://127.0.0")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func StartHealthServer() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		isHealthy := true
		failureMessage := ""
		currentMode := config.Instance.HealthMode

		logger.Debug("HEALTH", fmt.Sprintf("Executing active evaluation loop under strategy level code: %d", currentMode))

		if currentMode == config.LevelDisabled {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintln(w, "OK")
			return
		}

		// 1. Process Core Storage Shares Matrix
		for name, share := range TrackedShares {
			if currentMode == config.LevelNfs && name != "nfs" {
				logger.Debug("HEALTH", fmt.Sprintf("Skipping component evaluation: share [%s] does not match target [nfs] mode filter.", name))
				continue
			}
			if currentMode == config.LevelSamba && name != "samba" {
				logger.Debug("HEALTH", fmt.Sprintf("Skipping component evaluation: share [%s] does not match target [samba] mode filter.", name))
				continue
			}

			err := share.Healthcheck()
			if err != nil {
				shouldFail := false

				switch currentMode {
				case config.LevelFull, config.LevelShares:
					logger.Debug("HEALTH", fmt.Sprintf("Switch trace [Full/Shares]: component failure on [%s] forces an automatic container error.", name))
					shouldFail = true
				case config.LevelCritical, config.LevelDefault:
					logger.Debug("HEALTH", fmt.Sprintf("Switch trace [Critical/Default]: evaluating importance flag for component [%s] (IsCritical: %v)", name, share.IsCritical()))
					shouldFail = share.IsCritical()
				case config.LevelNfs, config.LevelSamba:
					logger.Debug("HEALTH", fmt.Sprintf("Switch trace [Nfs/Samba]: targeted protocol failure on [%s] forces an automatic container error.", name))
					shouldFail = true
				}

				if shouldFail {
					isHealthy = false
					failureMessage = fmt.Sprintf("Critical storage share error on component [%s]", name)
					logger.Error("HEALTH", failureMessage, err)
					break
				} else {
					logger.Error("HEALTH", fmt.Sprintf("Muted non-critical share process issue on component [%s]", name), err)
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
						logger.Debug("HEALTH", fmt.Sprintf("Switch trace [Full]: beacon failure on [%s] forces an automatic container error.", name))
						shouldFail = true
					case config.LevelCritical:
						logger.Debug("HEALTH", fmt.Sprintf("Switch trace [Critical]: evaluating importance flag for beacon [%s] (IsCritical: %v)", name, beacon.IsCritical()))
						shouldFail = beacon.IsCritical()
					}

					if shouldFail {
						isHealthy = false
						failureMessage = fmt.Sprintf("Critical advertisement beacon error on component [%s]", name)
						logger.Error("HEALTH", failureMessage, err)
						break
					} else {
						logger.Error("HEALTH", fmt.Sprintf("Muted non-critical beacon notification fault on component [%s]", name), err)
					}
				}
			}
		}

		if !isHealthy {
			logger.Debug("HEALTH", fmt.Sprintf("Writing HTTP payload response status: 503 Service Unavailable -> Content: FAIL: %s", failureMessage))
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, "FAIL: %s\n", failureMessage)
			return
		}

		logger.Debug("HEALTH", "Writing HTTP payload response status: 200 OK -> Content: OK")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	})

	go func() {
		if err := http.ListenAndServe("127.0.0.1:8080", nil); err != nil {
			logger.Error("HEALTH", "Internal loop HTTP health server tracking daemon collapsed", err)
		}
	}()
}
