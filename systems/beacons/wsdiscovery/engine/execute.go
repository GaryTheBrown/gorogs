package engine

import (
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/incoming"
	"gorogs/systems/beacons/wsdiscovery/templates"
	"gorogs/systems/beacons/wsdiscovery/versions"
)

func (s *Engine) ExecuteProbeAction(msg incoming.WSMessage) {
	senderString := msg.Sender.String()

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ProbeMatches.String(),
		msg.Header.MessageID,
		msg.Header.ReplyToURL,
	)
	if err != nil {
		logger.Error(Name, "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	if msg.UseTCPTransport && msg.Header.ReplyToURL != "" {
		err = connection.SendTCPUnicastResponse(payloadBytes, msg.Header.ReplyToURL)
		if err != nil {
			logger.ErrorF(Name, "[TCP Engine] HTTP POST transaction delivery failed for client target: %s", err, msg.Header.ReplyToURL)
			return
		}
		return
	}

	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.ErrorF(Name, "Socket transmission delivery failed for network target endpoint: %s", err, senderString)
		return
	}
}

func (s *Engine) ExecuteResolveAction(msg incoming.WSMessage) {
	senderString := msg.Sender.String()

	payloadBytes, err := templates.GenerateXMLResponse(
		msg.SchemaVersion,
		versions.ToValueList[msg.SchemaVersion]["reply"],
		versions.SchemaList[msg.SchemaVersion][versions.Discovery],
		versions.ResolveMatches.String(),
		msg.Header.MessageID,
		msg.Header.ReplyToURL,
	)
	if err != nil {
		logger.Error(Name, "XML text transformation engine crashed processing assets folder templates", err)
		return
	}

	if msg.UseTCPTransport && msg.Header.ReplyToURL != "" {
		err = connection.SendTCPUnicastResponse(payloadBytes, msg.Header.ReplyToURL)
		if err != nil {
			logger.ErrorF(Name, "[TCP Engine] HTTP POST transaction delivery failed for client target: %s", err, msg.Header.ReplyToURL)
			return
		}
		return
	}

	err = connection.SendUnicastResponse(payloadBytes, msg.Sender)
	if err != nil {
		logger.ErrorF(Name, "Socket transmission delivery failed for network target endpoint: %s", err, senderString)
		return
	}
}

func (s *Engine) ExecuteGetAction(msg incoming.WSMessage) {
	if msg.HTTPResponsePipe == nil {
		logger.Error(Name, "ExecuteGetAction dropped execution pass: missing transaction channel reference", nil)
		return
	}

	senderString := msg.Sender.String()
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
		logger.ErrorF(Name, "[TCP Engine] Failed to generate XML GetResponse metadata context block for host: %s", err, senderString)
		msg.HTTPResponsePipe <- connection.HTTPResponsePayload{Err: err}
		return
	}
	msg.HTTPResponsePipe <- connection.HTTPResponsePayload{BodyBytes: xmlPayload}
}
