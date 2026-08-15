package incoming

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"gorogs/beacons/wsdd/versions"
	"gorogs/logger"
)

var InstanceUUID string
var EnableFastDecoding bool

func Decode(rawUDP []byte, message *WSMessage) error {
	logger.Debug("wsdd", "Initializing new list collection structure matrices")
	lists := NewListsStruct()

	logger.Debug("wsdd", fmt.Sprintf("Commencing outer SOAP envelope wrapper unmarshalling pass for %d bytes", len(rawUDP)))
	var envelope SoapEnvelope
	decoder := xml.NewDecoder(bytes.NewReader(rawUDP))
	if err := decoder.Decode(&envelope); err != nil {
		logger.Error("wsdd", "Outer SOAP frame wrapper deserialization pass failed", err)
		return ErrBadSchemaUnmarshalFailed{ExternalError: err}
	}

	logger.Debug("wsdd", "Parsing structural message header parameter elements")
	if err := ParseHeader(&envelope, message); err != nil {
		logger.Error("wsdd", "Header parser function threw structural verification exception", err)
		return ErrBadSchemaFailedHeaderRead{ExternalError: err}
	}

	if !EnableFastDecoding && message.Header.ActionType != versions.Get {
		logger.Debug("wsdd", fmt.Sprintf("Executing gatekeeper envelope attribute strict check for version: %s", message.SchemaVersion))
		if err := ValidateStrictNamespaces(message.SchemaVersion, envelope.Attrs, lists); err != nil {
			logger.Error("wsdd", "Strict envelope attribute namespace gatekeeper pass rejected payload layout", err)
			return err
		}

		logger.Debug("wsdd", "Commencing full document deep tag token validation scan traversal pass")
		if err := ValidateFullDocumentNamespaces(message.SchemaVersion, rawUDP, lists); err != nil {
			logger.Error("wsdd", "Full document tag name token validator scan discovered unauthorized schema configuration", err)
			return err
		}
	}

	actionTypeString := message.Header.ActionType.String()
	logger.Info("wsdd", fmt.Sprintf("Full document structure validated successfully. Extracting body payload type: %s", actionTypeString))

	switch message.Header.ActionType {
	case versions.Probe:
		if err := ParseBodyProbe(envelope.Body.RawInner, message); err != nil {
			logger.Error("wsdd", "Parsing logic failed extracting parameters from Probe inner body block", err)
			return err
		}
	case versions.Resolve:
		logger.Info("wsdd", "PARSING RESOLVE MESSAGE")
		if err := ParseBodyResolve(envelope.Body.RawInner, message); err != nil {
			errCheck := ErrResolveNotForUs{}
			if err != errCheck {
				logger.Error("wsdd", "Parsing logic failed extracting parameters from Resolve inner body block", err)
			}
			return err
		}
	case versions.Hello:
		if err := ParseBodyHello(envelope.Body.RawInner, message); err != nil {
			logger.Error("wsdd", "Parsing logic failed extracting parameters from Hello inner body block", err)
			return err
		}
	case versions.Bye:
		if err := ParseBodyBye(envelope.Body.RawInner, message); err != nil {
			logger.Error("wsdd", "Parsing logic failed extracting parameters from Bye inner body block", err)
			return err
		}
	case versions.GetMetadata:
		if err := ParseBodyGetMetadata(envelope.Body.RawInner, message); err != nil {
			logger.Error("wsdd", "Parsing logic failed extracting parameters from GetMetadata inner body block", err)
			return err
		}
	default:
		logger.Debug("wsdd", fmt.Sprintf("No unique structural body unmarshaler mapped for execution criteria type: %s", actionTypeString))
	}

	logger.Debug("wsdd", "Parsing tracking process successfully concluded for raw payload block")
	return nil
}

func ParseHeader(envelope *SoapEnvelope, message *WSMessage) error {
	actionStr := strings.TrimSpace(envelope.Header.Action)
	if actionStr == "" {
		err := errors.New("missing mandatory Action header")
		logger.Error("wsdd", "Header verification failed", err)
		return err
	}

	lastSlash := strings.LastIndex(actionStr, "/")
	if lastSlash == -1 {
		err := errors.New("Action header is not a valid structured URL path")
		logger.Error("wsdd", fmt.Sprintf("Malformed context found evaluating header value string: '%s'", actionStr), err)
		return err
	}

	actionURL := actionStr[:lastSlash]
	actionCmdStr := actionStr[lastSlash+1:]

	logger.Debug("wsdd", fmt.Sprintf("Converting text tag component string token context target: '%s'", actionCmdStr))
	var err error
	message.Header.ActionType, err = versions.StringToActionType(actionCmdStr)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Action command string token mapping failed completely for payload action: '%s'", actionCmdStr), err)
		return err
	}

	if message.Header.ActionType == versions.Get {
		if actionURL != versions.TransferSchema {
			err = ErrTransferSchemaWrong{}
			logger.Error("wsdd", fmt.Sprintf("Transfer Schema Is not what we expect we got: %s", actionURL), err)
			return err
		}

		detectedVersion, found := GlobalSessionTracker.LookupVersion(message.Sender)
		if found {
			message.SchemaVersion = detectedVersion
			logger.Debug("wsdd", fmt.Sprintf("Successfully restored active protocol version session mapping from incoming cache: %s", message.SchemaVersion))
		} else {
			message.SchemaVersion = "2005/04"
			logger.Debug("wsdd", "No matching peer session context found inside incoming tracker. Assigning fallback version: 2005/04")
		}

	} else {
		logger.Debug("wsdd", fmt.Sprintf("Running matrix reverse lookup validating active discovery schema version identifier path matching: '%s'", actionURL))

		message.SchemaVersion, err = versions.SchemaList.CheckDiscoveryVersion(actionURL)
		if err != nil {
			logger.Error("wsdd", fmt.Sprintf("Active version check matrix rejected discovery path mapping context target: '%s'", actionURL), err)
			return err
		}

		GlobalSessionTracker.RecordSession(message.Sender, message.SchemaVersion)
	}

	message.Header.MessageID = envelope.Header.MessageID
	message.Header.To = envelope.Header.To
	message.Header.AppSequence.InstanceID = envelope.Header.AppSequence.InstanceId
	message.Header.AppSequence.MessageNumber = envelope.Header.AppSequence.MessageNumber

	logger.Info("wsdd", fmt.Sprintf("Header configuration extracted completely. MsgID: %s, Detected Protocol Specification Release: %s", message.Header.MessageID, message.SchemaVersion))
	return nil
}

func ParseBodyProbe(rawData []byte, message *WSMessage) error {
	logger.Debug("wsdd", fmt.Sprintf("Executing inner unmarshal operation targeting inner Probe body raw data block sizes: %d bytes", len(rawData)))
	var payload ProbePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		logger.Error("wsdd", "Probe structural body schema translation mapping task encountered fatal syntax error", err)
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Probe.Types = payload.Types
	logger.Debug("wsdd", fmt.Sprintf("Probe data block extracted cleanly. Target Capability Types parsed matching: %s", payload.Types))
	return nil
}

func ParseBodyResolve(rawData []byte, message *WSMessage) error {
	logger.Debug("wsdd", fmt.Sprintf("Executing inner unmarshal operation targeting inner Resolve body raw data block sizes: %d bytes", len(rawData)))
	var payload ResolvePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		logger.Error("wsdd", "Probe structural body schema translation mapping task encountered fatal syntax error", err)
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}

	requestUUID := strings.TrimPrefix(payload.EndpointReference.Address, "urn:uuid:")
	if InstanceUUID != requestUUID {
		logger.Info("wsdd", fmt.Sprintf("Resolve request not for us Dropping message expected=%s got=%s", InstanceUUID, requestUUID))
		return ErrResolveNotForUs{}
	}
	message.Body.Resolve.Address = requestUUID
	logger.Debug("wsdd", fmt.Sprintf("Probe data block extracted cleanly. Address Requesting: %s", payload.EndpointReference.Address))
	logger.Info("wsdd", "PARSED RESOLVE MESSAGE")
	return nil
}

func ParseBodyHello(rawData []byte, message *WSMessage) error {
	logger.Debug("wsdd", fmt.Sprintf("Executing inner unmarshal operation targeting inner Hello body raw data block sizes: %d bytes", len(rawData)))
	var payload HelloPayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		logger.Error("wsdd", "Hello structural body schema translation mapping task encountered fatal syntax error", err)
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Hello.Address = payload.EndpointReference.Address
	message.Body.Hello.Types = payload.Types
	logger.Debug("wsdd", fmt.Sprintf("Hello data block extracted cleanly. Device Address: %s, Announced Types: %s", payload.EndpointReference.Address, payload.Types))
	return nil
}

func ParseBodyBye(rawData []byte, message *WSMessage) error {
	logger.Debug("wsdd", fmt.Sprintf("Executing inner unmarshal operation targeting inner Bye body raw data block sizes: %d bytes", len(rawData)))
	var payload ByePayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		logger.Error("wsdd", "Bye structural body schema translation mapping task encountered fatal syntax error", err)
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.Bye.Address = payload.EndpointReference.Address

	logger.Debug("wsdd", fmt.Sprintf("Bye data block extracted cleanly. Device Disconnection Target Address: %s", payload.EndpointReference.Address))
	return nil
}

func ParseBodyGetMetadata(rawData []byte, message *WSMessage) error {
	logger.Debug("wsdd", fmt.Sprintf("Executing inner unmarshal operation targeting inner GetMetadata body raw data block sizes: %d bytes", len(rawData)))
	var payload GetMetadataPayload
	if err := xml.Unmarshal(rawData, &payload); err != nil {
		logger.Error("wsdd", "GetMetadata structural body schema translation mapping task encountered fatal syntax error", err)
		return ErrBadSchemaBodyUnmarshalFailed{ExternalError: err}
	}
	message.Body.GetMetadata.XMLName = payload.XMLName
	logger.Debug("wsdd", fmt.Sprintf("GetMetadata schema container header element parsed cleanly. Namespace Local Target tag: %s", payload.XMLName.Local))
	return nil
}
