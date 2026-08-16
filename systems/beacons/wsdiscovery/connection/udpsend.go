package connection

import (
	"fmt"
	"gorogs/logger"
	"net"
)

func SendUnicastResponse(payload []byte, targetAddr net.Addr) error {
	targetString := targetAddr.String()
	logger.Debug("WSDiscovery", fmt.Sprintf("Preparing outbound unicast response pipeline for remote host: %s", targetString))

	udpAddr, err := net.ResolveUDPAddr("udp4", targetString)
	if err != nil {
		logger.Error("WSDiscovery", fmt.Sprintf("Failed to resolve destination net.Addr structure into valid IPv4 address for host: %s", targetString), err)
		return err
	}

	if UDPConn != nil {
		if logger.DebugActive {
			logger.DebugF("WSDiscovery", "Reusing central socket to transmit unicast response out of port 3702 to: %s", udpAddr.String())
		}
		n, err := UDPConn.WriteToUDP(payload, udpAddr)
		if err != nil {
			logger.ErrorF("WSDiscovery", "Network write operation failure via central socket: %s", err, targetString)
			return err
		}
		logger.InfoF("WSDiscovery", "Successfully dispatched %d bytes from source port 3702 to client: %s", n, targetString)
		return nil
	}

	return fmt.Errorf("cannot send unicast response: central SharedConn socket is uninitialized")
}

func SendMulticastBroadcast(payload []byte) error {
	logger.DebugF("WSDiscovery", "Preparing outbound multicast protocol transmission loop for address target: %s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort)

	multicastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort))
	if err != nil {
		logger.Error("WSDiscovery", "Failed to resolve protocol specification multicast target address mapping rules", err)
		return err
	}

	if UDPConn != nil {
		logger.DebugF("WSDiscovery", "Reusing central socket to write multicast broadcast to group: %s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort)
		n, err := UDPConn.WriteToUDP(payload, multicastAddr)
		if err != nil {
			logger.Error("WSDiscovery", "Network write operation failure: data frame dropped writing multicast payload", err)
			return err
		}
		logger.InfoF("WSDiscovery", "Successfully broadcasted %d bytes over multicast channel to: %s:%s", n, DiscoveryMulticastIP, DiscoveryMulticastPort)
		return nil
	}

	return fmt.Errorf("cannot send multicast broadcast: central SharedConn socket is uninitialized")
}
