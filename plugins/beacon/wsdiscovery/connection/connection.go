package connection

import (
	"fmt"
	"gorogs/logger"
	"gorogs/plugins/beacon/wsdiscovery/incoming"

	"net"
	"net/http"

	"golang.org/x/net/ipv4"
)

const (
	DiscoveryMulticastIP   = "239.255.255.250"
	DiscoveryMulticastPort = "3702"
	TransferTCPPort        = "5357"
)

var (
	FastDecodingMode bool
)

var (
	UDPConn     *net.UDPConn
	TCPListener net.Listener
	Name        string
)

func InitUDPSocket() error {
	localAddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+DiscoveryMulticastPort)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp4", localAddr)
	if err != nil {
		return err
	}
	UDPConn = conn

	multicastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort))
	if err == nil {
		pConn := ipv4.NewPacketConn(UDPConn)
		if err := pConn.SetMulticastLoopback(false); err != nil {
			return err
		}
		if ifaces, err := net.Interfaces(); err == nil {
			for _, iface := range ifaces {
				if (iface.Flags&net.FlagUp) != 0 && (iface.Flags&net.FlagMulticast) != 0 {
					_ = pConn.JoinGroup(&iface, multicastAddr)
				}
			}
		}
	}

	return nil
}

func InitTCPSocket(discoveryQueue chan<- incoming.WSMessage) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		HandleIncomingHTTPTransfer(w, r, discoveryQueue)
	})

	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+TransferTCPPort, nil); err != nil {
			logger.Error(Name, "TCP server encountered a critical error", err)
		}
	}()

	return nil
}
