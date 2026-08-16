package templates

import (
	"bytes"
	"fmt"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/versions"

	"strings"
	"sync/atomic"
	"text/template"
	"time"
)

var CombinedTemplateCache = make(map[string]map[string]*template.Template)

type StaticBakingContext struct {
	ServerName         string
	Workgroup          string
	HostIP             string
	DomainSuffix       string
	InstanceID         uint64
	InstanceUUID       string
	MetadataVersion    uint32
	EnvelopeAttributes string
	ActionType         string
	BodyPayload        string
	SchemaSlice        []string
}

type RuntimeContext struct {
	To            string
	MessageID     string
	RelatesTo     string
	MessageNumber uint64
}

var schemaIndexMap map[string]int
var currentInstanceID uint64
var messageCounter atomic.Uint64

func getNextMessageNumber() uint64 {
	return messageCounter.Add(1)
}

func init() {
	currentInstanceID = uint64(time.Now().Unix())

	schemaIndexMap = make(map[string]int, int(versions.MaxSchemaType))
	for i := range int(versions.MaxSchemaType) {
		enumVal := versions.SchemaTypeEnum(i)
		prefixName := enumVal.String()
		if prefixName != "unknown" {
			schemaIndexMap[prefixName] = i
			schemaIndexMap[strings.ToLower(prefixName)] = i
		}
	}
}

func LookupSchema(ctx StaticBakingContext, schemaKey string) string {
	lowerKey := strings.ToLower(schemaKey)
	idx, exists := schemaIndexMap[lowerKey]
	if !exists {
		return ""
	}
	if idx < 0 || idx >= len(ctx.SchemaSlice) {
		return ""
	}
	return ctx.SchemaSlice[idx]
}

func BuildEnvelopeAttributes(schemaArray versions.SchemaListArray) string {
	var sb strings.Builder
	for i := range int(versions.MaxSchemaType) {
		schemaType := versions.SchemaTypeEnum(i)
		prefixName := schemaType.String()
		urlValue := schemaArray[schemaType]
		if prefixName == "unknown" || urlValue == "" {
			continue
		}
		if sb.Len() > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString("xmlns:")
		sb.WriteString(prefixName)
		sb.WriteString(`="`)
		sb.WriteString(urlValue)
		sb.WriteByte('"')
	}
	return sb.String()
}

func GenerateXMLResponse(currentVersion string, to string, actionurl string, action string, trackingMsgID string) ([]byte, error) {
	versionMap, foundVersion := CombinedTemplateCache[currentVersion]
	if !foundVersion {
		return nil, fmt.Errorf("requested schema version entry layout map matrix not found: %s", currentVersion)
	}

	tmpl, foundAction := versionMap[action]
	if !foundAction {
		return nil, fmt.Errorf("requested action token template layout not found in version sub-map: %s", action)
	}

	ctx := RuntimeContext{
		To:            to,
		MessageID:     GenerateRandomUUIDv4(),
		RelatesTo:     trackingMsgID,
		MessageNumber: getNextMessageNumber(),
	}

	var finalBuf bytes.Buffer
	if err := tmpl.Execute(&finalBuf, &ctx); err != nil {
		logger.Error("WSDiscovery", "Single-pass context rendering tracking pipeline encountered an execution failure", err)
		return nil, fmt.Errorf("failed to execute combined multi-version template payload: %w", err)
	}

	return finalBuf.Bytes(), nil
}
