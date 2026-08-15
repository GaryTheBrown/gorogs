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

var ShareRoot = "/srv"

var (
	Name         string
	DomainSuffix string
	ContainerIP  net.IP
	HostGateway  net.IP
	IsCheckMode  bool
	HealthMode   HealthLevel

	NfsEnabled         bool
	SambaEnabled       bool
	RpcbindEnabled     bool
	NmbdEnabled        bool
	WsddEnabled        bool
	MdnsEnabled        bool
	MdnsNfsEnabled     bool
	MdnsSambaEnabled   bool
	LiveChangesEnabled bool
	ZeroSpaceEnabled   bool
)
