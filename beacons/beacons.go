package beacons

import (
	"errors"
	"net"
)

// ErrServiceDisabled is returned during Setup to indicate a service is intentionally muted via env toggles
var ErrServiceDisabled = errors.New("service is disabled via environment configuration parameters")

type DiscoveryBeacon interface {
	Setup(config AppConfig) error
	Start() error
	Healthcheck() error
	IsCritical() bool
	//Requires() []string
	//LogSetup() <- how to do this or is this the right area for it should it be in setup?
	Stop() error
}

type AppConfig struct {
	ServerName   string
	DomainSuffix string
	ContainerIP  net.IP
	HostGateway  net.IP
}
