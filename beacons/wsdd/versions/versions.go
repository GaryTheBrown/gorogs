package versions

import (
	"errors"
	"fmt"
	"gorogs/logger"
)

var ErrVersionNotFound = errors.New("Version not found")
var ErrActionNotFound = errors.New("Action not found")
var ErrSchemaNotFound = errors.New("Schema not found")

type SchemaTypeEnum uint8

const (
	Soap SchemaTypeEnum = iota
	Addressing
	Discovery
	Mex
	DevProf
	PnPX
	Pub
	// MaxSchemaType must stay at the bottom.
	MaxSchemaType
	UNKNOWNSCHEMA = MaxSchemaType
)

type SchemaListArray [MaxSchemaType]string

var schemaNamesArray = [MaxSchemaType]string{
	Soap:       "soap",
	Addressing: "wsa",
	Discovery:  "wsd",
	Mex:        "wsx",
	DevProf:    "wsdp",
	PnPX:       "un0",
	Pub:        "pub",
}

func (st SchemaTypeEnum) String() string {
	if st >= MaxSchemaType {
		return "unknown"
	}
	return schemaNamesArray[st]
}

type ActionTypeEnum uint

const (
	Hello ActionTypeEnum = iota
	Bye
	Probe
	ProbeMatches
	Resolve
	ResolveMatches
	GetMetadata
	GetMetadataResponse
	Get
	GetResponse
	// MaxActionType must stay at the bottom.
	MaxActionType
	UNKNOWNACTION = MaxActionType
)

var actionNamesArray = [MaxActionType]string{
	Hello:               "Hello",
	Bye:                 "Bye",
	Probe:               "Probe",
	ProbeMatches:        "ProbeMatches",
	Resolve:             "Resolve",
	ResolveMatches:      "ResolveMatches",
	GetMetadata:         "GetMetadata",
	GetMetadataResponse: "GetMetadataResponse",
	Get:                 "Get",
	GetResponse:         "GetResponse",
}

var stringToActionMap map[string]ActionTypeEnum

func init() {
	stringToActionMap = make(map[string]ActionTypeEnum, MaxActionType)
	for i, name := range actionNamesArray {
		stringToActionMap[name] = ActionTypeEnum(i)
	}
}

func (at ActionTypeEnum) String() string {
	if at >= MaxActionType {
		return "unknown"
	}
	return actionNamesArray[at]
}

func StringToActionType(str string) (ActionTypeEnum, error) {
	action, found := stringToActionMap[str]
	if !found {
		logger.Error("wsdd", fmt.Sprintf("Action translation failure: target text command string '%s' does not exist in protocol vocabulary maps", str), ErrActionNotFound)
		return MaxActionType, ErrActionNotFound
	}
	return action, nil
}

func (sla SchemaListArray) Action(action ActionTypeEnum) (fullSchemaURL string, err error) {
	switch action {
	case Hello, Bye, Probe, ProbeMatches:
		return sla[Discovery] + "/" + action.String(), nil
	case GetMetadata, GetMetadataResponse:
		return sla[Mex] + "/" + action.String(), nil
	default:
		logger.Error("wsdd", fmt.Sprintf("Action schema resolution failure: target enum index id '%d' falls outside parsing matrix parameters", action), ErrActionNotFound)
		return "", ErrActionNotFound
	}
}

func (sla SchemaListArray) Find(schemaItemIn string) (schemaType SchemaTypeEnum, err error) {
	for i, schemaTypeItem := range sla {
		if schemaItemIn == schemaTypeItem {
			return SchemaTypeEnum(i), nil
		}
	}
	return MaxSchemaType, ErrSchemaNotFound
}

type SchemaListMap map[string]SchemaListArray

func (sl SchemaListMap) CheckDiscoveryVersion(i string) (schemaVersion string, err error) {
	for k := range sl {
		if i == sl[k][Discovery] {
			return k, nil
		}
	}
	logger.Error("wsdd", fmt.Sprintf("Discovery schema check failure: incoming target namespace URL path context '%s' does not match any registered specification configuration release tier", i), ErrVersionNotFound)
	return "", ErrVersionNotFound
}

func (slt SchemaListMap) Find(schemaItemIn string) (schemaVersion string, schemaType SchemaTypeEnum, err error) {
	for schemaVersion, schemaListItem := range slt {
		schemaType, err = schemaListItem.Find(schemaItemIn)
		if err == nil {
			return schemaVersion, schemaType, nil
		}
	}
	return "", MaxSchemaType, ErrSchemaNotFound
}

func (slt SchemaListMap) FindSkippingVersion(schemaItemIn string, skipVersin string) (schemaVersion string, schemaType SchemaTypeEnum, err error) {
	for schemaVersion, schemaListItem := range slt {
		if schemaVersion == skipVersin {
			continue
		}
		schemaType, err = schemaListItem.Find(schemaItemIn)
		if err == nil {
			return schemaVersion, schemaType, nil
		}
	}
	return "", MaxSchemaType, ErrSchemaNotFound
}
