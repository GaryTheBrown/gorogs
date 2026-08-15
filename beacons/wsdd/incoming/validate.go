package incoming

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"

	"gorogs/beacons/wsdd/versions"
	"gorogs/logger"
)

func ValidateStrictNamespaces(version string, attrs []xml.Attr, lists *listsStruct) error {
	schemaList, exists := versions.SchemaList[version]
	if !exists {
		logger.ErrorF("wsdd", "Namespace validator aborted: active schema release lookup matrix has no map matching release string version: '%s'", nil, version)
		return ErrVersionNotFound{}
	}

	logger.DebugF("wsdd", "Auditing envelope attributes. Validating %d items against active vocabulary registration rules.", len(attrs))

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
					logger.ErrorF("wsdd", "Gatekeeper pass violation: detected duplicate namespace prefix target schema type registration mapping for prefix: '%s'", nil, attr.Name.Local)
					return ErrDuplicateNamespacePrefix{}
				}
				if logger.IsDebugActive("wsdd") {
					logger.DebugF("wsdd", "Valid namespace matched: Attribute Prefix Key '%s' binds cleanly to explicit Schema Standard Type [%s]", attr.Name.Local, schemaType.String())
				}
				lists.Found[schemaType] = true
				continue
			}
			if err == versions.ErrSchemaNotFound {
				_, _, err2 := versions.SchemaList.FindSkippingVersion(urlVal, version)
				if err2 == versions.ErrSchemaNotFound {
					_, foundInList := lists.UnknownSchema[prefixKey]
					if foundInList {
						logger.ErrorF("wsdd", "Gatekeeper pass violation: detected duplicate unknown prefix schema target mapping inside registration lists for prefix: '%s'", nil, prefixKey)
						return ErrDuplicateNamespacePrefix{}
					}
					logger.DebugF("wsdd", "Registering extension/external unmanaged vendor namespace path shortcut link: %s=\"%s\"", prefixKey, urlVal)
					lists.UnknownSchema[prefixKey] = urlVal
					continue
				}

				logger.ErrorF("wsdd", "Gatekeeper pass violation: schema URL matches a completely different WS-Discovery version variant than current runtime target configuration expects. Rejected URL: %s", nil, urlVal)
				return ErrBadSchemaWrongVersion{}
			} else {
				lists.Shortcut[prefixKey] = schemaType
				continue
			}
		}
	}

	logger.Info("wsdd", "Strict gatekeeper pass successfully completed. All envelope namespace attributes comply with protocol definitions.")
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
	logger.Debug("wsdd", fmt.Sprintf("Commencing complete token scanner deep pass for raw packet array segment size: %d bytes", len(rawXML)))
	decoder := xml.NewDecoder(bytes.NewReader(rawXML))

	for {
		tok, err := decoder.Token()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			logger.Error("wsdd", "Structural stream failure: low-level XML token token reader parser stream errored mid-scan", err)
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
			if expectedSpace == name.Space {
				logger.DebugF("wsdd", "[Token Match] Verified element <%s:%s> correctly binds to expected active protocol namespace path URL: %s", schemaType.String(), name.Local, name.Space)
				continue
			} else {
				logger.ErrorF("wsdd", "[Schema Mismatch] Element variant structural validation failure: tag name '%s' attempted to execute using unauthorized namespace URL space: %s", nil, name.Local, name.Space)
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
				logger.ErrorF("wsdd", "[Unknown Tag Error] Security verification pass rejected unrecognized or unapproved rogue payload tag name element target: <%s:%s>", nil, name.Space, name.Local)
				return ErrBadSchemaTagNameBad{}
			}
			logger.DebugF("wsdd", "[Extension Tag Match] Processing safe vendor extension tag name component: <%s:%s>", name.Space, name.Local)
		}
	}

	logger.Info("wsdd", "Full document tag name token scanner deep pass safely completed with zero violations detected.")
	return nil
}
