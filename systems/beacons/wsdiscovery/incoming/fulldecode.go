package incoming

import (
	"bytes"
	"encoding/xml"
	"errors"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/versions"
	"strings"
)

var InstanceUUID string
var SkipValidation bool

func FullDecode(rawUDP []byte, message *WSMessage) error {
	lists := NewListsStruct()
	var envelope SoapEnvelope
	decoder := xml.NewDecoder(bytes.NewReader(rawUDP))
	if err := decoder.Decode(&envelope); err != nil {
		return ErrBadSchemaUnmarshalFailed{ExternalError: err}
	}

	logger.Debug(Name, "Parsing structural message header parameter elements")
	if err := ParseHeader(&envelope, message); err != nil {
		return ErrBadSchemaFailedHeaderRead{ExternalError: err}
	}

	if !SkipValidation && message.Header.ActionType != versions.Get {
		if err := ValidateStrictNamespaces(message.SchemaVersion, envelope.Attrs, lists); err != nil {
			return err
		}
		if err := ValidateFullDocumentNamespaces(message.SchemaVersion, rawUDP, lists); err != nil {
			return err
		}
	}

	switch message.Header.ActionType {
	case versions.Probe:
		if err := ParseBodyProbe(envelope.Body.RawInner, message); err != nil {
			return err
		}
	case versions.Resolve:
		if err := ParseBodyResolve(envelope.Body.RawInner, message); err != nil {
			return err
		}
	case versions.Hello:
		if err := ParseBodyHello(envelope.Body.RawInner, message); err != nil {
			return err
		}
	case versions.Bye:
		if err := ParseBodyBye(envelope.Body.RawInner, message); err != nil {
			return err
		}
	case versions.GetMetadata:
		if err := ParseBodyGetMetadata(envelope.Body.RawInner, message); err != nil {
			return err
		}
	default:
	}

	return nil
}

func ParseHeader(envelope *SoapEnvelope, message *WSMessage) error {
	actionStr := strings.TrimSpace(envelope.Header.Action)
	if actionStr == "" {
		err := errors.New("missing mandatory Action header")
		return err
	}

	lastSlash := strings.LastIndex(actionStr, "/")
	if lastSlash == -1 {
		err := errors.New("Action header is not a valid structured URL path")
		return err
	}

	actionURL := actionStr[:lastSlash]
	actionCmdStr := actionStr[lastSlash+1:]

	var err error
	message.Header.ActionType, err = versions.StringToActionType(actionCmdStr)
	if err != nil {
		return err
	}

	if message.Header.ActionType == versions.Get {
		if actionURL != versions.TransferSchema {
			err = ErrTransferSchemaWrong{}
			return err
		}

		detectedVersion, found := GlobalSessionTracker.LookupVersion(message.Sender)
		if found {
			message.SchemaVersion = detectedVersion
		} else {
			message.SchemaVersion = "2005/04"
		}

	} else {
		message.SchemaVersion, err = versions.SchemaList.CheckDiscoveryVersion(actionURL)
		if err != nil {
			return err
		}

		GlobalSessionTracker.RecordSession(message.Sender, message.SchemaVersion)
	}

	message.Header.MessageID = envelope.Header.MessageID
	message.Header.To = envelope.Header.To
	message.Header.AppSequence.InstanceID = envelope.Header.AppSequence.InstanceId
	message.Header.AppSequence.MessageNumber = envelope.Header.AppSequence.MessageNumber

	if envelope.Header.ReplyTo != nil && envelope.Header.ReplyTo.Address != "" {
		rawAddr := strings.TrimSpace(envelope.Header.ReplyTo.Address)

		if strings.HasPrefix(rawAddr, "http://") || strings.HasPrefix(rawAddr, "https://") {
			message.Header.ReplyToURL = rawAddr
			message.UseTCPTransport = true
		}
	}
	return nil
}

func ParseBodyProbe(rawData []byte, message *WSMessage) error {
	var payload ProbePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Probe.Types = payload.Types
	return nil
}

func ParseBodyResolve(rawData []byte, message *WSMessage) error {
	var payload ResolvePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}

	requestUUID := strings.TrimPrefix(payload.EndpointReference.Address, "urn:uuid:")
	if InstanceUUID != requestUUID {
		return ErrResolveNotForUs{}
	}
	message.Body.Resolve.Address = requestUUID
	return nil
}

func ParseBodyHello(rawData []byte, message *WSMessage) error {
	var payload HelloPayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Hello.Address = payload.EndpointReference.Address
	message.Body.Hello.Types = payload.Types
	return nil
}

func ParseBodyBye(rawData []byte, message *WSMessage) error {
	var payload ByePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Bye.Address = payload.EndpointReference.Address

	return nil
}

func ParseBodyGetMetadata(rawData []byte, message *WSMessage) error {
	var payload GetMetadataPayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.GetMetadata.XMLName = payload.XMLName
	return nil
}
