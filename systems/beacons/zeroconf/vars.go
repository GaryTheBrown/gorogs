package zeroconf

import (
	"fmt"
	"iter"
	"strings"
)

type AddrType struct {
	Enabled bool
	Name    string
	Address string
	Port    uint16
}

func (at *AddrType) Disable() {
	at.Enabled = false
}

func (at *AddrType) Enable() {
	at.Enabled = true
}

type AddrMap map[string]AddrType

func (am *AddrMap) Add(name string, port uint16) {
	Addrs[name] = AddrType{
		Enabled: true,
		Name:    name,
		Address: fmt.Sprintf("_%s.%s", name, tcpLocal),
		Port:    port,
	}
}

const (
	ttlUnique = uint32(120)
	ttlShared = uint32(4500)
	ttlLegacy = uint32(10)
	ttlProbe  = uint32(255)
	ttlBye    = uint32(0)
)

var (
	hostTarget         string
	hostParam          string
	tcpLocal           string
	udpLocal           string
	servicesMetaRecord string
	serverIcon         string
	serverName         string

	Addrs          AddrMap
	txtRecordsList []string

	activebroadcaster    bool
	forceLocalDomainName bool
	forceNoDomainName    bool
	singleTextRecord      bool

	disabledSlice iter.Seq[string]
	enabledSlice  iter.Seq[string]
)

func init() {
	Addrs = AddrMap{}

	txtRecordsList = []string{
		"path=/",
		fmt.Sprintf("host=%s", hostParam),
		fmt.Sprintf("model=%s", serverIcon),
	}
}

func AddrSetup() {

	Addrs.Add("smb", 445)
	Addrs.Add("nfs", 2049)

	for val := range disabledSlice {
		cleanVal := strings.TrimSpace(val)
		if addr, ok := Addrs[cleanVal]; ok {
			addr.Disable()
			Addrs[cleanVal] = addr
		}
	}

	for val := range enabledSlice {
		cleanVal := strings.TrimSpace(val)
		if addr, ok := Addrs[cleanVal]; ok {
			addr.Enable()
			Addrs[cleanVal] = addr
		}
	}
}

var ()
