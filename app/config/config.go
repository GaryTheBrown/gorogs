package config

import "net"

type HealthLevel int

const (
	LevelDefault HealthLevel = iota
	LevelFull
	LevelCritical
	LevelShares
	LevelNfs
	LevelSamba
	LevelDisabled
)

// ShareRoot holds the absolute master storage path for the entire appliance
const ShareRoot = "/srv"

type HubConfig struct {
	Name         string
	DomainSuffix string
	ContainerIP  net.IP
	HostGateway  net.IP
	IsCheckMode  bool
	HealthMode   HealthLevel

	NfsEnabled         bool
	SambaEnabled       bool
	RpcbindEnabled     bool
	WsddEnabled        bool
	MdnsEnabled        bool
	LiveChangesEnabled bool
}

var Instance *HubConfig
