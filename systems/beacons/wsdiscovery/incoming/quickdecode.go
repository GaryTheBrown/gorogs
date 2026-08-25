package incoming

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"gorogs/systems/beacons/wsdiscovery/versions"
)

func QuickDecode(rawUDP []byte, message *WSMessage) error {
	actionBytes, err := extractTagContents(rawUDP, []byte("<wsa:Action>"), []byte("</wsa:Action>"))
	if err != nil {
		actionBytes, err = extractTagContents(rawUDP, []byte("<Action>"), []byte("</Action>"))
		if err != nil {
			return errors.New("quickdecode: missing mandatory Action header boundary token")
		}
	}
	actionStr := string(bytes.TrimSpace(actionBytes))

	lastSlash := strings.LastIndex(actionStr, "/")
	if lastSlash == -1 {
		return errors.New("quickdecode: Action header is not a valid structured URL path")
	}
	actionURL := actionStr[:lastSlash]
	actionCmdStr := actionStr[lastSlash+1:]

	message.Header.ActionType, err = versions.StringToActionType(actionCmdStr)
	if err != nil {
		return fmt.Errorf("quickdecode: unknown action command token mapping target: %s", actionCmdStr)
	}

	if message.Header.ActionType == versions.Get {
		if actionURL != versions.TransferSchema {
			return ErrTransferSchemaWrong{}
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

	message.Header.MessageID = string(extractTagContentsOrDefault(rawUDP, []byte("<wsa:MessageID>"), []byte("</wsa:MessageID>")))
	if message.Header.MessageID == "" {
		message.Header.MessageID = string(extractTagContentsOrDefault(rawUDP, []byte("<MessageID>"), []byte("</MessageID>")))
	}
	message.Header.MessageID = strings.TrimPrefix(message.Header.MessageID, "urn:uuid:")

	message.Header.To = string(extractTagContentsOrDefault(rawUDP, []byte("<wsa:To>"), []byte("</wsa:To>")))
	if message.Header.To == "" {
		message.Header.To = string(extractTagContentsOrDefault(rawUDP, []byte("<To>"), []byte("</To>")))
	}

	replyToBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsa:ReplyTo>"), []byte("</wsa:ReplyTo>"))
	if len(replyToBytes) == 0 {
		replyToBytes = extractTagContentsOrDefault(rawUDP, []byte("<ReplyTo>"), []byte("</ReplyTo>"))
	}

	if len(replyToBytes) > 0 {
		addrStrBytes := extractTagContentsOrDefault(replyToBytes, []byte("<wsa:Address>"), []byte("</wsa:Address>"))
		if len(addrStrBytes) == 0 {
			addrStrBytes = extractTagContentsOrDefault(replyToBytes, []byte("<Address>"), []byte("</Address>"))
		}

		if len(addrStrBytes) > 0 {
			rawAddrURL := string(addrStrBytes)
			if strings.HasPrefix(rawAddrURL, "http://") || strings.HasPrefix(rawAddrURL, "https://") {
				message.Header.ReplyToURL = rawAddrURL
				message.UseTCPTransport = true
			}
		}
	}

	if message.Header.ActionType == versions.Get {
		return nil
	}

	return executeQuickBodyParsing(rawUDP, message)
}

func extractTagContents(data []byte, startTag, endTag []byte) ([]byte, error) {
	startIdx := bytes.Index(data, startTag)
	if startIdx == -1 {
		return nil, errors.New("tag not found")
	}
	startIdx += len(startTag)

	endIdx := bytes.Index(data[startIdx:], endTag)
	if endIdx == -1 {
		return nil, errors.New("closing tag not found")
	}

	return data[startIdx : startIdx+endIdx], nil
}

func extractTagContentsOrDefault(data []byte, startTag, endTag []byte) []byte {
	res, err := extractTagContents(data, startTag, endTag)
	if err != nil {
		return nil
	}
	return bytes.TrimSpace(res)
}

func executeQuickBodyParsing(rawUDP []byte, message *WSMessage) error {
	switch message.Header.ActionType {
	case versions.Probe:
		typesBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsd:Types>"), []byte("</wsd:Types>"))
		if len(typesBytes) == 0 {
			typesBytes = extractTagContentsOrDefault(rawUDP, []byte("<Types>"), []byte("</Types>"))
		}
		message.Body.Probe.Types = string(typesBytes)

	case versions.Resolve:
		addrBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsa:Address>"), []byte("</wsa:Address>"))
		if len(addrBytes) == 0 {
			addrBytes = extractTagContentsOrDefault(rawUDP, []byte("<Address>"), []byte("</Address>"))
		}
		message.Body.Resolve.Address = strings.TrimPrefix(string(addrBytes), "urn:uuid:")

	case versions.Hello:
		addrBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsa:Address>"), []byte("</wsa:Address>"))
		if len(addrBytes) == 0 {
			addrBytes = extractTagContentsOrDefault(rawUDP, []byte("<Address>"), []byte("</Address>"))
		}
		message.Body.Hello.Address = strings.TrimPrefix(string(addrBytes), "urn:uuid:")

		typesBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsd:Types>"), []byte("</wsd:Types>"))
		message.Body.Hello.Types = string(typesBytes)

	case versions.Bye:
		addrBytes := extractTagContentsOrDefault(rawUDP, []byte("<wsa:Address>"), []byte("</wsa:Address>"))
		if len(addrBytes) == 0 {
			addrBytes = extractTagContentsOrDefault(rawUDP, []byte("<Address>"), []byte("</Address>"))
		}
		message.Body.Bye.Address = strings.TrimPrefix(string(addrBytes), "urn:uuid:")

	default:

	}

	return nil
}
