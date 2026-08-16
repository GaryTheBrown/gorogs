package health

import (
	"fmt"
	"net/http"

	"gorogs/logger"
)

func (h *CheckStruct) handler(w http.ResponseWriter, r *http.Request) {
	isHealthy := true
	failureMessage := ""

	logger.DebugF(logName, "Executing active evaluation loop under strategy level code: %d", h.healthMode)

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
				logger.ErrorF(logName, "Critical storage share error on component [%s]", err, name)
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
					logger.ErrorF(logName, "Critical advertisement beacon error on component [%s]", err, name)
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
}
