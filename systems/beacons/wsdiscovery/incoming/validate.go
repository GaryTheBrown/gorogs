package incoming

import (
	"bytes"
	"encoding/xml"
	"strings"

	"gorogs/systems/beacons/wsdiscovery/versions"
)

func ValidateStrictNamespaces(version string, attrs []xml.Attr, lists *listsStruct) error {
	schemaList, exists := versions.SchemaList[version]
	if !exists {
		return ErrVersionNotFound{}
	}

	for _, attr := range attrs {
		if attr.Name.Space == "xmlns" || (attr.Name.Local == "xmlns" && attr.Name.Space == "") {
			prefixKey := attr.Name.Local
			if attr.Name.Local == "xmlns" && attr.Name.Space == "" {
				prefixKey = "$default"
			}

			urlVal := strings.TrimSpace(attr.Value)
			schemaType, err := schemaList.Find(urlVal)
			if err == nil {
				if lists.Found[schemaType] {
					return ErrDuplicateNamespacePrefix{}
				}
				lists.Found[schemaType] = true
				continue
			}
			if err == versions.ErrSchemaNotFound {
				_, _, err2 := versions.SchemaList.FindSkippingVersion(urlVal, version)
				if err2 == versions.ErrSchemaNotFound {
					_, foundInList := lists.UnknownSchema[prefixKey]
					if foundInList {
						return ErrDuplicateNamespacePrefix{}
					}
					lists.UnknownSchema[prefixKey] = urlVal
					continue
				}

				return ErrBadSchemaWrongVersion{}
			} else {
				lists.Shortcut[prefixKey] = schemaType
				continue
			}
		}
	}

	return nil
}

var approvedMap = map[string]versions.SchemaTypeEnum{
	"Envelope":          versions.Soap,
	"Header":            versions.Soap,
	"Body":              versions.Soap,
	"To":                versions.Addressing,
	"Action":            versions.Addressing,
	"MessageID":         versions.Addressing,
	"EndpointReference": versions.Addressing,
	"Address":           versions.Addressing,
	"ReplyTo":           versions.Addressing,
	"AppSequence":       versions.Discovery,
	"MetadataVersion":   versions.Discovery,
	"XAddrs":            versions.Discovery,
	"Probe":             versions.Discovery,
	"Resolve":           versions.Discovery,
	"Types":             versions.Discovery,
	"Scopes":            versions.Discovery,
	"Hello":             versions.Discovery,
	"Bye":               versions.Discovery,
}

func ValidateFullDocumentNamespaces(version string, rawXML []byte, lists *listsStruct) error {
	decoder := xml.NewDecoder(bytes.NewReader(rawXML))

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return ErrBadSchemaDecoderFailed{ExternalError: err}
		}
		var name xml.Name

		if se, ok := tok.(xml.StartElement); ok {
			name = se.Name
		} else if ee, ok := tok.(xml.EndElement); ok {
			name = ee.Name
		} else {
			continue
		}

		if schemaType, exists := approvedMap[name.Local]; exists {
			expectedSpace := versions.SchemaList[version][schemaType]

			isCrossVersionAddressingMatch := false
			if schemaType == versions.Addressing {
				for _, schemaArray := range versions.SchemaList {
					if schemaArray[versions.Addressing] == name.Space {
						isCrossVersionAddressingMatch = true
						break
					}
				}
			}
			if expectedSpace == name.Space || isCrossVersionAddressingMatch {
				continue
			} else {
				return ErrBadSchemaTagNameBad{}
			}

		} else {
			foundInUnknownList := false
			for _, v := range lists.UnknownSchema {
				if v == name.Space {
					foundInUnknownList = true
					break
				}
			}
			if !foundInUnknownList {
				return ErrBadSchemaTagNameBad{}
			}
		}
	}

	return nil
}
