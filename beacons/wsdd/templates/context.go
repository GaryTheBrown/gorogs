package templates

import (
	"bytes"
	"fmt"
	"gorogs/beacons"
	"gorogs/beacons/wsdd/versions"
	"gorogs/logger"
	"strings"
	"sync/atomic"
	"text/template"
	"time"
)

type AppSeqContext struct {
}

type TemplateContext struct {
	SchemaSlice        []string
	EnvelopeAttributes string
	To                 string
	ActionType         string
	MessageID          string
	RelatesTo          string
	BodyPayload        string
	InstanceID         uint64
	MessageNumber      uint64
	InstanceUUID       string
	MetadataVersion    uint32
	ServerName         string
	Workgroup          string
	HostIP             string
	DomainSuffix       string
}

var schemaIndexMap map[string]int
var currentInstanceID uint64
var messageCounter uint64

func getNextMessageNumber() uint64 {
	return atomic.AddUint64(&messageCounter, 1)
}

func init() {
	currentInstanceID = uint64(time.Now().Unix())
	schemaIndexMap = make(map[string]int, int(versions.MaxSchemaType))
	for i := 0; i < int(versions.MaxSchemaType); i++ {
		enumVal := versions.SchemaTypeEnum(i)
		prefixName := enumVal.String()
		if prefixName != "unknown" {
			schemaIndexMap[prefixName] = i
			schemaIndexMap[strings.ToLower(prefixName)] = i
		}
	}
}

func LookupSchema(ctx TemplateContext, schemaKey string) string {
	lowerKey := strings.ToLower(schemaKey)
	idx, exists := schemaIndexMap[lowerKey]
	if !exists {
		logger.Debug("wsdd", fmt.Sprintf("Template schema lookup skipped: key '%s' not registered in schema index map", schemaKey))
		return ""
	}
	if idx < 0 || idx >= len(ctx.SchemaSlice) {
		logger.Error("wsdd", fmt.Sprintf("Template schema index out of bounds: resolved index %d for key '%s' fails slice capacity bounds check", idx, schemaKey), nil)
		return ""
	}
	return ctx.SchemaSlice[idx]
}

func BuildEnvelopeAttributes(schemaArray versions.SchemaListArray) string {
	var sb strings.Builder
	for i := 0; i < int(versions.MaxSchemaType); i++ {
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

func GenerateXMLResponse(currentVersion string, to string, actionurl string, action string, trackingMsgID string, config beacons.AppConfig, instanceUUID string) ([]byte, error) {
	logger.Debug("wsdd", fmt.Sprintf("Generating XML payload configuration contexts for current action type match target: %s", action))
	activeSchemaArray := versions.SchemaList[currentVersion]
	bodyFilename := "body" + strings.ToLower(action)

	logger.Debug("wsdd", fmt.Sprintf("Retrieving baked minified buffer segment representation text data for key reference: %s", bodyFilename))
	bodyTemplateText, err := LoadTemplate(bodyFilename)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Fatal template pipeline read interruption: cannot resolve file template component text for: %s", bodyFilename), err)
		return nil, err
	}
	headerTemplateText, err := LoadTemplate("header")
	if err != nil {
		logger.Error("wsdd", "Fatal template pipeline read interruption: cannot resolve structural envelope header template component text data", err)
		return nil, err
	}
	ctx := TemplateContext{
		SchemaSlice:        activeSchemaArray[:],
		EnvelopeAttributes: BuildEnvelopeAttributes(activeSchemaArray),
		To:                 to,
		ActionType:         actionurl + "/" + action,
		MessageID:          GenerateRandomUUIDv4(),
		RelatesTo:          trackingMsgID,
		InstanceID:         currentInstanceID,
		MessageNumber:      getNextMessageNumber(),
		InstanceUUID:       instanceUUID,
		MetadataVersion:    1,
		ServerName:         config.ServerName,
		Workgroup:          "WORKGROUP",
		HostIP:             config.ContainerIP.String(),
		DomainSuffix:       config.DomainSuffix,
	}

	if strings.ToLower(action) == "bye" {
		ctx.MetadataVersion = 2
	}

	logger.Debug("wsdd", fmt.Sprintf("Outbound message runtime metadata structures provisioned successfully. Assigned fresh temporary transactional ID: %s", ctx.MessageID))
	funcMap := template.FuncMap{
		"schema": LookupSchema,
		"upper":  strings.ToUpper,
		"lower":  strings.ToLower,
	}
	bodyTmpl, err := template.New("body").Funcs(funcMap).Parse(bodyTemplateText)
	if err != nil {
		logger.Error("wsdd", "Template initialization syntax compilation check failed on inner body structures", err)
		return nil, fmt.Errorf("failed to parse body template: %w", err)
	}
	var bodyBuf bytes.Buffer
	if err := bodyTmpl.Execute(&bodyBuf, &ctx); err != nil {
		logger.Error("wsdd", "Data rendering processing failure rendering variable context bindings to raw inner body template", err)
		return nil, fmt.Errorf("failed to execute body template: %w", err)
	}
	ctx.BodyPayload = bodyBuf.String()
	headerTmpl, err := template.New("header").Funcs(funcMap).Parse(headerTemplateText)
	if err != nil {
		logger.Error("wsdd", "Template initialization syntax compilation check failed on outer global header structural layouts", err)
		return nil, fmt.Errorf("failed to parse header template: %w", err)
	}
	var finalBuf bytes.Buffer
	if err := headerTmpl.Execute(&finalBuf, &ctx); err != nil {
		logger.Error("wsdd", "Data rendering processing failure rendering nested variable context boundaries to main envelope header framework template", err)
		return nil, fmt.Errorf("failed to execute final envelope: %w", err)
	}
	logger.Info("wsdd", fmt.Sprintf("XML data rendering successfully completed. Generated %d bytes compliant buffer ready to send onto network wire", finalBuf.Len()))
	return finalBuf.Bytes(), nil
}
