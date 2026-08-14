package incoming

import (
	"encoding/xml"
	"net"

	"gorogs/beacons/wsdd/versions"
)

// --------------
// Tracking of schema struct
// --------------
type listsStruct struct {
	Found         [versions.MaxSchemaType]bool
	Shortcut      map[string]versions.SchemaTypeEnum
	UnknownSchema map[string]string
}

func (l *listsStruct) Reset() {
	for i := range l.Found {
		l.Found[i] = false
	}
	for k := range l.Shortcut {
		delete(l.Shortcut, k)
	}
	for k := range l.UnknownSchema {
		delete(l.UnknownSchema, k)
	}
}

func NewListsStruct() *listsStruct {
	return &listsStruct{
		Shortcut:      make(map[string]versions.SchemaTypeEnum),
		UnknownSchema: make(map[string]string),
	}
}

// ---------------
// xml output struct
// ---------------
type WSMessage struct {
	Sender        net.Addr
	SchemaVersion string

	HTTPResponsePipe chan interface{}

	Header struct {
		To          string
		ActionType  versions.ActionTypeEnum
		MessageID   string
		AppSequence struct {
			InstanceID    int64
			MessageNumber int64
		}
	}
	Body struct {
		Probe struct {
			Types string
		}
		Hello struct {
			Address string
			Types   string
		}
		Bye struct {
			Address string
		}
		GetMetadata struct {
			XMLName xml.Name
		}
		Resolve struct {
			Address string
		}
	}
}

// ---------
// XML STUFF
// ---------
type SoapEnvelope struct {
	XMLName xml.Name   `xml:"Envelope"`
	Attrs   []xml.Attr `xml:",attr"`
	Header  struct {
		MessageID   string `xml:"MessageID"`
		Action      string `xml:"Action"`
		To          string `xml:"To"`
		AppSequence struct {
			InstanceId    int64 `xml:"InstanceId,attr"`
			MessageNumber int64 `xml:"MessageNumber,attr"`
		} `xml:"AppSequence"`
	} `xml:"Header"`
	Body struct {
		RawInner []byte `xml:",innerxml"`
	} `xml:"Body"`
}

type EndpointReference struct {
	Address string `xml:"Address"`
}

type ProbePayload struct {
	XMLName xml.Name `xml:"Probe"`
	Types   string   `xml:"Types"`
	Scopes  string   `xml:"Scopes"`
}

type HelloPayload struct {
	XMLName           xml.Name          `xml:"Hello"`
	EndpointReference EndpointReference `xml:"EndpointReference"`
	Types             string            `xml:"Types"`
	Scopes            string            `xml:"Scopes"`
	XAddrs            string            `xml:"XAddrs"`
	MetadataVersion   int64             `xml:"MetadataVersion"`
}

type ByePayload struct {
	XMLName           xml.Name          `xml:"Bye"`
	EndpointReference EndpointReference `xml:"EndpointReference"`
}

type ResolvePayload struct {
	XMLName           xml.Name          `xml:"Resolve"`
	EndpointReference EndpointReference `xml:"EndpointReference"`
}

type GetMetadataPayload struct {
	XMLName xml.Name `xml:"GetMetadata"`
}
