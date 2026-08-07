package beacons

import "errors"

// ErrServiceDisabled is returned during Setup to indicate a service is intentionally muted via env toggles
var ErrServiceDisabled = errors.New("service is disabled via environment configuration parameters")

type DiscoveryBeacon interface {
	Setup() error
	Start() error
	Healthcheck() error
	IsCritical() bool
	Stop() error
}
