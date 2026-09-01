package healthcheck

import (
	"fmt"
	"net/http"
	"strings"

	"gorogs/logger"
	"gorogs/plugins/share/nfs"
	"gorogs/plugins/share/samba"
	"gorogs/system/systeminterface"
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
		if healthMode == Nfs && !strings.EqualFold(name, nfs.Name) {
			continue
		}
		if healthMode == Samba && !strings.EqualFold(name, samba.Name) {
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
				shouldFail = strings.EqualFold(name, nfs.Name)
			case Samba:
				shouldFail = strings.EqualFold(name, samba.Name)
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
