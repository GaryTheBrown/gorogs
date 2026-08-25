package healthcheck

import (
	"fmt"
	"net/http"
	"strings"

	"gorogs/logger"
	"gorogs/systems/shares/nfs"
	"gorogs/systems/shares/samba"
	"gorogs/systems/systeminterface"
)

func handler(w http.ResponseWriter, r *http.Request) {
	logger.DebugF(logName, "Executing active evaluation loop under strategy level code: %d", healthMode)
	isHealthy := true
	failureMessage := ""
	if healthMode == Disabled {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
		return
	}

	for name, share := range tracked {
		if healthMode == Nfs && name != strings.ToLower(nfs.Name) {
			continue
		}
		if healthMode == Samba && name != strings.ToLower(samba.Name) {
			continue
		}

		err := share.Healthcheck()
		if err != nil {
			shouldFail := false
			switch healthMode {
			case Full:
				shouldFail = true
			case Critical:
				shouldFail = share.IsCritical()
			case Shares:
				shouldFail = (systeminterface.Share == share.Type())
			case Nfs:
				shouldFail = (name == strings.ToLower(nfs.Name))
			case Samba:
				shouldFail = (name == strings.ToLower(samba.Name))
			case Default:
				shouldFail = (systeminterface.Share == share.Type())
			}

			if shouldFail {
				isHealthy = false
				logger.ErrorF(logName, "Critical storage share error on component [%s]", err, name)
				break
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
