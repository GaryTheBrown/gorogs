package helpers

import (
	"net"
	"strings"
	"time"
)

func WaitForSocket(network, address string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if strings.HasPrefix(network, "udp") {
			packetConn, err := net.ListenPacket(network, address)
			if err != nil {
				if strings.Contains(err.Error(), "address already in use") {
					return true
				}
			} else {
				packetConn.Close()
			}
		} else {
			conn, err := net.DialTimeout(network, address, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return true
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return false
}
