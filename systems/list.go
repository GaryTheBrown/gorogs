package systems

import (
	"gorogs/systems/beacons/netbios"
	"gorogs/systems/beacons/rpcbind"
	"gorogs/systems/beacons/wsdiscovery"
	"gorogs/systems/beacons/zeroconf"
	"gorogs/systems/shares/nfs"
	"gorogs/systems/shares/samba"
	"gorogs/systems/systeminterface"
	"gorogs/systems/utilities/zerospace"
)

// THIS EVENTUALLY NEEDS TO BE GENERATED AUTOMATICALLY BY READING THE SYSTEMS AND QUERYING THEM THE ORDER
type SystemNameEnum uint16

const (
	//ORDER IS IMPORTANT DON'T CHANGE THIS ORDER
	ZeroSpace SystemNameEnum = iota
	RpcBind
	NFS
	NetBIOS
	Samba
	WSDiscovery
	ZeroCONF
)

var systemList = []systeminterface.System{
	//ORDER IS IMPORTANT DON'T CHANGE THIS ORDER
	&zerospace.Struct{},
	&rpcbind.Struct{},
	&nfs.Struct{},
	&netbios.Struct{},
	&samba.Struct{},
	&wsdiscovery.Struct{},
	&zeroconf.Struct{},
}
