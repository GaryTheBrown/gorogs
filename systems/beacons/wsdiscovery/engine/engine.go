package engine

import (
	"context"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/incoming"
	"gorogs/systems/beacons/wsdiscovery/templates"
	"gorogs/systems/beacons/wsdiscovery/versions"
)

type Engine struct {
	DiscoveryQueue chan incoming.WSMessage
	ListenerDone   <-chan struct{}
	ServiceDone    chan struct{}
}

func NewEngineState() *Engine {
	return &Engine{
		DiscoveryQueue: make(chan incoming.WSMessage, 100),
		ServiceDone:    make(chan struct{}),
	}
}

func (s *Engine) Start(ctx context.Context, configDir string) error {

	templates.LoadOrCreatePersistentUUID(configDir, config.Hostname)

	logger.InfoF("WSDiscovery", "Configuring centralized "+connection.DiscoveryMulticastPort+" UDP socket infrastructure parameters")
	if err := connection.InitUDPSocket(); err != nil {
		logger.Error("WSDiscovery", "Fatal break: Central WS-Discovery UDP socket failed to bind", err)
		close(s.DiscoveryQueue)
		return err
	}

	logger.InfoF("WSDiscovery", "Configuring centralized %s TCP socket infrastructure parameters", connection.TransferTCPPort)
	if err := connection.InitTCPSocket(s.DiscoveryQueue); err != nil {
		logger.Error("WSDiscovery", "Fatal break: WS-Transfer TCP socket failed to bind", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}

	logger.InfoF("WSDiscovery", "Initializing engine multicast routing on interface address: %s", config.SystemIP)
	doneChan, err := connection.UDPListener(ctx, s.DiscoveryQueue)
	if err != nil {
		logger.Error("WSDiscovery", "Failed to construct low-level network reader socket", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}
	s.ListenerDone = doneChan

	logger.Info("WSDiscovery", "Launching background action dispatcher worker queue thread loop")
	go s.ActionDispatcher(ctx)

	s.BroadcastHello()

	return nil
}

func (s *Engine) ActionDispatcher(ctx context.Context) {
	defer close(s.ServiceDone)
	logger.Info("WSDiscovery", "Action dispatcher processing loop successfully listening for network events")

	for msg := range s.DiscoveryQueue {
		actionName := msg.Header.ActionType.String()
		senderString := msg.Sender.String()

		logger.DebugF("WSDiscovery", "Processing packet dequeued from channel. Type: %s, Source: %s", actionName, senderString)
		switch msg.Header.ActionType {
		case versions.Probe:
			s.ExecuteProbeAction(msg)
		case versions.Resolve:
			s.ExecuteResolveAction(msg)
		case versions.Get:
			s.ExecuteGetAction(msg)
		default:
			logger.DebugF("WSDiscovery", "Skipping operational command handler logic for action category type: %s", actionName)
		}
	}

	logger.Info("WSDiscovery", "Discovery channel closed. Waiting for multicast listener socket cleanup window...")
	<-s.ListenerDone
	logger.Info("WSDiscovery", "Background action loop thread completely drained and shut down.")
}

func (s *Engine) ExecuteProbeAction(msg incoming.WSMessage) {
	senderString := msg.Sender.String()
	logger.InfoF("WSDiscovery", "Matching capabilities matrix for client search probe from: %s", senderString)

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ProbeMatches.String(),
		msg.Header.MessageID,
		msg.Header.ReplyToURL,
	)
	if err != nil {
		logger.Error("WSDiscovery", "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	// 🚀 NEW: Dual-Transport Output Router Routing Bridge
	if msg.UseTCPTransport && msg.Header.ReplyToURL != "" {
		logger.DebugF("WSDiscovery", "[TCP Engine] Transmitting compiled byte payload size %d via HTTP POST to: %s", len(payloadBytes), msg.Header.ReplyToURL)
		err = connection.SendTCPUnicastResponse(payloadBytes, msg.Header.ReplyToURL)
		if err != nil {
			logger.ErrorF("WSDiscovery", "[TCP Engine] HTTP POST transaction delivery failed for client target: %s", err, msg.Header.ReplyToURL)
			return
		}
		logger.InfoF("WSDiscovery", "[TCP Engine] Successfully dispatched ProbeMatches framework to: %s", msg.Header.ReplyToURL)
		return
	}

	// Fallback to traditional UDP Unicast path
	logger.DebugF("WSDiscovery", "Flash transmitting compiled minified byte payload size %d via unicast to client: %s", len(payloadBytes), senderString)
	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.ErrorF("WSDiscovery", "Socket transmission delivery failed for network target endpoint: %s", err, senderString)
		return
	}
	logger.InfoF("WSDiscovery", "Successfully dispatched complete ProbeMatches response framework to: %s", senderString)
}

func (s *Engine) ExecuteResolveAction(msg incoming.WSMessage) {
	senderString := msg.Sender.String()
	logger.InfoF("WSDiscovery", "Matching capabilities matrix for client search Resolve from: %s", senderString)

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ResolveMatches.String(),
		msg.Header.MessageID,
		msg.Header.ReplyToURL,
	)
	if err != nil {
		logger.Error("WSDiscovery", "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	// 🚀 NEW: Dual-Transport Output Router Routing Bridge
	if msg.UseTCPTransport && msg.Header.ReplyToURL != "" {
		logger.DebugF("WSDiscovery", "[TCP Engine] Transmitting compiled byte payload size %d via HTTP POST to: %s", len(payloadBytes), msg.Header.ReplyToURL)
		err = connection.SendTCPUnicastResponse(payloadBytes, msg.Header.ReplyToURL)
		if err != nil {
			logger.ErrorF("WSDiscovery", "[TCP Engine] HTTP POST transaction delivery failed for client target: %s", err, msg.Header.ReplyToURL)
			return
		}
		logger.InfoF("WSDiscovery", "[TCP Engine] Successfully dispatched ResolveMatches framework to: %s", msg.Header.ReplyToURL)
		return
	}

	// Fallback to traditional UDP Unicast path
	logger.DebugF("WSDiscovery", "Flash transmitting compiled minified byte payload size %d via unicast to client: %s", len(payloadBytes), senderString)
	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.ErrorF("WSDiscovery", "Socket transmission delivery failed for network target endpoint: %s", err, senderString)
		return
	}
	logger.InfoF("WSDiscovery", "Successfully dispatched complete ResolveMatches response framework to: %s", senderString)
}

func (s *Engine) ExecuteGetAction(msg incoming.WSMessage) {
	if msg.HTTPResponsePipe == nil {
		logger.Error("WSDiscovery", "ExecuteGetAction dropped execution pass: missing transaction channel reference", nil)
		return
	}

	senderString := msg.Sender.String()
	logger.DebugF("WSDiscovery", "[TCP Engine] Processing metadata rendering pass for client connection: %s using version path context: %s", senderString, msg.SchemaVersion)

	anonymousTarget := versions.ToValueList[msg.SchemaVersion]["reply"]

	xmlPayload, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		anonymousTarget,
		versions.TransferSchema,
		"GetResponse",
		msg.Header.MessageID,
		"",
	)
	if err != nil {
		logger.ErrorF("WSDiscovery", "[TCP Engine] Failed to generate XML GetResponse metadata context block for host: %s", err, senderString)
		msg.HTTPResponsePipe <- connection.HTTPResponsePayload{Err: err}
		return
	}

	logger.DebugF("WSDiscovery", "[TCP Engine] Handing over %d payload bytes down to connection package sender framework", len(xmlPayload))

	msg.HTTPResponsePipe <- connection.HTTPResponsePayload{BodyBytes: xmlPayload}
	logger.InfoF("WSDiscovery", "[TCP Engine] Successfully satisfied metadata extraction handshake transaction loop with client: %s", senderString)
}

func (s *Engine) BroadcastHello() {
	logger.Info("WSDiscovery", "Commencing multi-version sequence broadcast for Hello advertisement pass...")
	for schemaVersion := range versions.SchemaList {
		logger.DebugF("WSDiscovery", "Compiling Hello advertisement template framework target version string: %s", schemaVersion)
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Hello.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF("WSDiscovery", "XML transmission synthesis failed on Hello announcement serialization steps for version: %s", err, schemaVersion)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF("WSDiscovery", "Multicast transmission delivery failed for Hello startup packet frame version: %s", err, schemaVersion)
			continue
		}
	}
	logger.Info("WSDiscovery", "Multi-version Hello announcement pass successfully broadcasted onto subnet.")
}

func (s *Engine) BroadcastBye() {
	logger.Info("WSDiscovery", "Commencing multi-version sequence broadcast for Bye shutdown pass...")
	for schemaVersion := range versions.SchemaList {
		logger.DebugF("WSDiscovery", "Compiling Bye notice template framework target version string: %s", schemaVersion)
		payloadBytes, err := templates.GenerateXMLResponse(
			schemaVersion,
			versions.ToValueList[schemaVersion]["request"],
			versions.SchemaList[schemaVersion][versions.Discovery],
			versions.Bye.String(),
			"",
			"",
		)
		if err != nil {
			logger.ErrorF("WSDiscovery", "XML transmission synthesis failed on Bye notice serialization steps for version: %s", err, schemaVersion)
			continue
		}
		err = connection.SendMulticastBroadcast(payloadBytes)
		if err != nil {
			logger.ErrorF("WSDiscovery", "Multicast transmission delivery failed for Bye shutdown packet frame version: %s", err, schemaVersion)
			continue
		}
	}
	logger.Info("WSDiscovery", "Multi-version Bye notice pass successfully broadcasted onto subnet.")
}

func (s *Engine) Stop() {
	logger.Info("WSDiscovery", "Signaling consumer queues to cease collection operations...")

	s.BroadcastBye()

	<-s.ServiceDone
	close(s.DiscoveryQueue)
	logger.Info("WSDiscovery", "Execution core processing loops fully closed.")
}
