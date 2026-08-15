package connection

import (
	"fmt"
	"gorogs/beacons/wsdd/incoming"
	"gorogs/logger"
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
)

func InitUDPSocket() error {
	logger.InfoF("wsdd", "Initializing centralized WS-Discovery socket on port: %s ", DiscoveryMulticastPort)

	localAddr, err := net.ResolveUDPAddr("udp4", "0.0.0.0:"+DiscoveryMulticastPort)
	if err != nil {
		logger.Error("wsdd", "Failed to resolve local UDP binding address", err)
		return err
	}

	conn, err := net.ListenUDP("udp4", localAddr)
	if err != nil {
		logger.ErrorF("wsdd", "Fatal error: Unable to bind to port %s. Is another daemon running?", err, DiscoveryMulticastPort)
		return err
	}
	UDPConn = conn

	multicastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort))
	if err == nil {
		pConn := ipv4.NewPacketConn(UDPConn)

		if err := pConn.SetMulticastLoopback(false); err != nil {
			logger.Error("wsdd", "Failed to apply multicast loopback suppression flag to connection packet layer", err)
		}

		if ifaces, err := net.Interfaces(); err == nil {
			for _, iface := range ifaces {
				if (iface.Flags&net.FlagUp) != 0 && (iface.Flags&net.FlagMulticast) != 0 {
					_ = pConn.JoinGroup(&iface, multicastAddr)
				}
			}
		}
	}

	logger.InfoF("wsdd", "Centralized WS-Discovery socket initialized and bound to port %s successfully", DiscoveryMulticastPort)
	return nil
}

func InitTCPSocket(discoveryQueue chan<- incoming.WSMessage) error {
	logger.InfoF("wsdd", "Initializing native WS-Transfer HTTP router on port %s", TransferTCPPort)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		HandleIncomingHTTPTransfer(w, r, discoveryQueue)
	})

	go func() {
		if err := http.ListenAndServe("0.0.0.0:"+TransferTCPPort, nil); err != nil {
			logger.Error("wsdd", "TCP server encountered a critical error", err)
		}
	}()

	logger.InfoF("wsdd", "WS-Transfer HTTP router online and listening on port %s successfully", TransferTCPPort)
	return nil
}
