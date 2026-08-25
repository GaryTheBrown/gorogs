package connection

import (
	"fmt"
	"gorogs/logger"
	"net"
)

func SendUnicastResponse(payload []byte, targetAddr net.Addr) error {
	targetString := targetAddr.String()

	udpAddr, err := net.ResolveUDPAddr("udp4", targetString)
	if err != nil {
		logger.Error(Name, fmt.Sprintf("Failed to resolve destination net.Addr structure into valid IPv4 address for host: %s", targetString), err)
		return err
	}

	if UDPConn != nil {
		_, err := UDPConn.WriteToUDP(payload, udpAddr)
		if err != nil {
			logger.ErrorF(Name, "Network write operation failure via central socket: %s", err, targetString)
			return err
		}
		return nil
	}

	return fmt.Errorf("cannot send unicast response: central SharedConn socket is uninitialized")
}

func SendMulticastBroadcast(payload []byte) error {

	multicastAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort))
	if err != nil {
		logger.Error(Name, "Failed to resolve protocol specification multicast target address mapping rules", err)
		return err
	}

	if UDPConn != nil {
		logger.DebugF(Name, "Reusing central socket to write multicast broadcast to group: %s:%s", DiscoveryMulticastIP, DiscoveryMulticastPort)
		_, err := UDPConn.WriteToUDP(payload, multicastAddr)
		if err != nil {
			logger.Error(Name, "Network write operation failure: data frame dropped writing multicast payload", err)
			return err
		}
		return nil
	}

	return fmt.Errorf("cannot send multicast broadcast: central SharedConn socket is uninitialized")
}
