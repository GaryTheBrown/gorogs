package config

var disableOptions = map[string]bool{
	//shares
	"nfs":   false,
	"samba": false,
	//beacons
	"rpcbind":     false,
	"netbios":     false,
	"wsdiscovery": false,
	"zeroconf":    false,
	"mdns_nfs":    false, //move to mdns options?
	"mdns_samba":  false, //move to mdns options?
	//Utilities
	"livechanges": false,
	"zerospace":   false,
}

var enableOptions = map[string]bool{
	//shares
	"nfs":   false,
	"samba": false,
	//beacons
	"rpcbind":     false,
	"netbios":     false,
	"wsdiscovery": false,
	"zeroconf":    false,
	//Utilities
	"livechanges": false,
	"zerospace":   false,
}
var (
	disabled map[string]bool
	enabled  map[string]bool
)

var massConfigMap = make(ConfigMap)
