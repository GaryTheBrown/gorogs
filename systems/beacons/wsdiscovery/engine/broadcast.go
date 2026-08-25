package engine

import (
	"bytes"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/templates"
	"gorogs/systems/beacons/wsdiscovery/versions"
	"net"
	"time"
)

func (s *Engine) BroadcastHello() {
	for schemaVersion := range versions.SchemaList {
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Hello.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF(Name, "XML transmission synthesis failed on Hello announcement serialization steps for version: %s", err, schemaVersion)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF(Name, "Multicast transmission delivery failed for Hello startup packet frame version: %s", err, schemaVersion)
			continue
		}
	}
}

func (s *Engine) BroadcastBye() {
	addr, err := net.ResolveUDPAddr("udp4", "239.255.255.250:3702")
	var listener *net.UDPConn
	if err == nil {
		listener, _ = net.ListenMulticastUDP("udp4", nil, addr)
	}
	if listener != nil {
		defer listener.Close()
	}

	for schemaVersion := range versions.SchemaList {
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Bye.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF(Name, "XML transmission synthesis failed on Bye notice serialization steps for version: %s", err, schemaVersion)
			continue
		}

		var packetCleared chan struct{}
		if listener != nil {
			packetCleared = make(chan struct{})

			go func(targetPayload []byte, confirmChan chan struct{}) {
				buffer := make([]byte, len(targetPayload)+256)
				for {
					_ = listener.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
					n, _, readErr := listener.ReadFrom(buffer)
					if readErr != nil {
						return
					}

					if bytes.Equal(buffer[:n], targetPayload) {
						close(confirmChan)
						return
					}
				}
			}(payloadBytes, packetCleared)
		}

		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF(Name, "Multicast transmission delivery failed for Bye shutdown packet frame version: %s", err, schemaVersion)
			continue
		}
		if packetCleared != nil {
			select {
			case <-packetCleared:
			case <-time.After(400 * time.Millisecond):
				logger.ErrorF(Name, "Inline verification timed out for version [%s]: Proceeding with graceful teardown.", nil, schemaVersion)
			}
		}
	}
}
