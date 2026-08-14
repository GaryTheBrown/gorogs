package engine

import (
	"context"
	"fmt"
	"gorogs/beacons/wsdd/connection"
	"gorogs/beacons/wsdd/incoming"
	"gorogs/beacons/wsdd/templates"
	"gorogs/beacons/wsdd/versions"
	"gorogs/logger"
)

type EngineState struct {
	DiscoveryQueue chan incoming.WSMessage
	ListenerDone   <-chan struct{}
	ServiceDone    chan struct{}
	ServerName     string
	HostIP         string
	InstanceUUID   string
}

func NewEngineState() *EngineState {
	return &EngineState{
		DiscoveryQueue: make(chan incoming.WSMessage, 100),
		ServiceDone:    make(chan struct{}),
	}
}

func StartEngine(ctx context.Context, s *EngineState, serverName, hostIP, configDir string) error {
	s.ServerName = serverName
	s.HostIP = hostIP
	s.InstanceUUID = templates.LoadOrCreatePersistentUUID(configDir, serverName)
	incoming.InstanceUUID = s.InstanceUUID

	logger.Info("wsdd", "Configuring centralized "+connection.DiscoveryMulticastPort+" UDP socket infrastructure parameters")
	if err := connection.InitUDPSocket(); err != nil {
		logger.Error("wsdd", "Fatal break: Central WS-Discovery UDP socket failed to bind", err)
		close(s.DiscoveryQueue)
		return err
	}

	logger.Info("wsdd", "Configuring centralized "+connection.TransferTCPPort+" TCP socket infrastructure parameters")
	if err := connection.InitTCPSocket(s.DiscoveryQueue); err != nil {
		logger.Error("wsdd", "Fatal break: WS-Transfer TCP socket failed to bind", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}

	logger.Info("wsdd", fmt.Sprintf("Initializing engine multicast routing on interface address: %s", s.HostIP))
	doneChan, err := connection.UDPListener(ctx, s.DiscoveryQueue)
	if err != nil {
		logger.Error("wsdd", "Failed to construct low-level network reader socket", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}
	s.ListenerDone = doneChan

	logger.Info("wsdd", "Launching background action dispatcher worker queue thread loop")
	go ActionDispatcher(ctx, s)

	BroadcastHello(s)

	return nil
}

func ActionDispatcher(ctx context.Context, s *EngineState) {
	defer close(s.ServiceDone)
	logger.Info("wsdd", "Action dispatcher processing loop successfully listening for network events")

	for msg := range s.DiscoveryQueue {
		actionName := msg.Header.ActionType.String()
		senderString := msg.Sender.String()

		logger.Debug("wsdd", fmt.Sprintf("Processing packet dequeued from channel. Type: %s, Source: %s", actionName, senderString))
		logger.Info("wsdd", fmt.Sprintf("ACTION DISPATCHER ABOUT TO DEAL WITH %s", msg.Header.ActionType.String()))
		switch msg.Header.ActionType {
		case versions.Probe:
			ExecuteProbeAction(s, msg)
		case versions.Resolve:
			ExecuteResolveAction(s, msg)
		case versions.Hello:
			logger.Info("wsdd", fmt.Sprintf("Observed Hello message announcement from external subnet node device: %s", senderString))
		case versions.Bye:
			logger.Info("wsdd", fmt.Sprintf("Observed Bye message disconnection notice from external subnet node device: %s", senderString))
		case versions.GetMetadata:
			logger.Info("wsdd", fmt.Sprintf("Received direct metadata schema configuration probe query request from: %s", senderString))
		case versions.Get:
			ExecuteGetAction(s, msg)
		default:
			logger.Debug("wsdd", fmt.Sprintf("Skipping operational command handler logic for action category type: %s", actionName))
		}
	}

	logger.Info("wsdd", "Discovery channel closed. Waiting for multicast listener socket cleanup window...")
	<-s.ListenerDone
	logger.Info("wsdd", "Background action loop thread completely drained and shut down.")
}

func ExecuteProbeAction(s *EngineState, msg incoming.WSMessage) {
	senderString := msg.Sender.String()
	logger.Info("wsdd", fmt.Sprintf("Matching capabilities matrix for client search probe from: %s", senderString))

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ProbeMatches.String(),
		msg.Header.MessageID,
		s.ServerName,
		s.HostIP,
		s.InstanceUUID,
	)
	if err != nil {
		logger.Error("wsdd", "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	logger.Debug("wsdd", fmt.Sprintf("Flash transmitting compiled minified byte payload size %d via unicast to client: %s", len(payloadBytes), senderString))
	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Socket transmission delivery failed for network target endpoint: %s", senderString), err)
		return
	}

	logger.Info("wsdd", fmt.Sprintf("Successfully dispatched complete ProbeMatches response framework to: %s", senderString))
}

func ExecuteResolveAction(s *EngineState, msg incoming.WSMessage) {
	logger.Info("wsdd", "EXECUTE RESOLVE ACTION")
	senderString := msg.Sender.String()
	logger.Info("wsdd", fmt.Sprintf("Matching capabilities matrix for client search Resolve from: %s", senderString))

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ResolveMatches.String(),
		msg.Header.MessageID,
		s.ServerName,
		s.HostIP,
		s.InstanceUUID,
	)
	if err != nil {
		logger.Error("wsdd", "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	logger.Debug("wsdd", fmt.Sprintf("Flash transmitting compiled minified byte payload size %d via unicast to client: %s", len(payloadBytes), senderString))
	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Socket transmission delivery failed for network target endpoint: %s", senderString), err)
		return
	}

	logger.Info("wsdd", fmt.Sprintf("Successfully dispatched complete ResolveMatches response framework to: %s", senderString))
}

func ExecuteGetAction(s *EngineState, msg incoming.WSMessage) {
	if msg.HTTPResponsePipe == nil {
		logger.Error("wsdd", "ExecuteGetAction dropped execution pass: missing transaction channel reference", nil)
		return
	}

	senderString := msg.Sender.String()
	logger.Debug("wsdd", fmt.Sprintf("[TCP Engine] Processing metadata rendering pass for client connection: %s using version path context: %s", senderString, msg.SchemaVersion))

	anonymousTarget := versions.ToValueList[msg.SchemaVersion]["reply"]

	xmlPayload, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		anonymousTarget,
		versions.TransferSchema,
		"GetResponse",
		msg.Header.MessageID,
		s.ServerName,
		s.HostIP,
		s.InstanceUUID,
	)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("[TCP Engine] Failed to generate XML GetResponse metadata context block for host: %s", senderString), err)
		msg.HTTPResponsePipe <- connection.HTTPResponsePayload{Err: err}
		return
	}

	logger.Debug("wsdd", fmt.Sprintf("[TCP Engine] Handing over %d payload bytes down to connection package sender framework", len(xmlPayload)))

	msg.HTTPResponsePipe <- connection.HTTPResponsePayload{BodyBytes: xmlPayload}
	logger.Info("wsdd", fmt.Sprintf("[TCP Engine] Successfully satisfied metadata extraction handshake transaction loop with client: %s", senderString))
}

func BroadcastHello(s *EngineState) {
	logger.Info("wsdd", "Commencing multi-version sequence broadcast for Hello advertisement pass...")
	for schemaVersion := range versions.SchemaList {
		logger.Debug("wsdd", fmt.Sprintf("Compiling Hello advertisement template framework target version string: %s", schemaVersion))
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Hello.String(),
			"",
			s.ServerName,
			s.HostIP,
			s.InstanceUUID,
		)
		if err != nil {
			logger.Error("wsdd", fmt.Sprintf("XML transmission synthesis failed on Hello announcement serialization steps for version: %s", schemaVersion), err)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.Error("wsdd", fmt.Sprintf("Multicast transmission delivery failed for Hello startup packet frame version: %s", schemaVersion), err)
			continue
		}
	}
	logger.Info("wsdd", "Multi-version Hello announcement pass successfully broadcasted onto subnet.")
}

func BroadcastBye(s *EngineState) {
	logger.Info("wsdd", "Commencing multi-version sequence broadcast for Bye shutdown pass...")
	for schemaVersion := range versions.SchemaList {
		logger.Debug("wsdd", fmt.Sprintf("Compiling Bye notice template framework target version string: %s", schemaVersion))
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Bye.String(),
			"",
			s.ServerName,
			s.HostIP,
			s.InstanceUUID,
		)
		if err != nil {
			logger.Error("wsdd", fmt.Sprintf("XML transmission synthesis failed on Bye notice serialization steps for version: %s", schemaVersion), err)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.Error("wsdd", fmt.Sprintf("Multicast transmission delivery failed for Bye shutdown packet frame version: %s", schemaVersion), err)
			continue
		}
	}
	logger.Info("wsdd", "Multi-version Bye notice pass successfully broadcasted onto subnet.")
}

func StopEngine(s *EngineState) {
	logger.Info("wsdd", "Signaling consumer queues to cease collection operations...")

	BroadcastBye(s)

	<-s.ServiceDone
	close(s.DiscoveryQueue)
	logger.Info("wsdd", "Execution core processing loops fully closed.")
}
