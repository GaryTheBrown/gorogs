package templates

import (
	"bytes"
	"fmt"
	"gorogs/config"
	"gorogs/plugins/beacon/wsdiscovery/versions"

	"io/fs"
	"strings"
	"text/template"
)

func PreCompileTemplates() error {
	preFuncMap := template.FuncMap{
		"schema": LookupSchema,
		"upper":  strings.ToUpper,
		"lower":  strings.ToLower,
	}

	runtimeFuncMap := template.FuncMap{
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
	}

	rawHeader, err := LoadTemplate("header")
	if err != nil {
		return fmt.Errorf("Fatal initialization break: Missing core envelope header asset template: %w", err)
	}

	entries, err := fs.ReadDir(xmlFS, "xml")
	if err != nil {
		return fmt.Errorf("Failed to scan dynamic template tokens from embedFS storage layer: %w", err)
	}

	var templateFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())

		if token, ok := strings.CutPrefix(name, "body"); ok {
			if idx := strings.Index(token, "."); idx != -1 {
				token = token[:idx]
			}
			templateFiles = append(templateFiles, token)
		}
	}

	for schemaVersion, schemaArray := range versions.SchemaList {
		CombinedTemplateCache[schemaVersion] = make(map[string]*template.Template)
		envelopeAttrsStr := BuildEnvelopeAttributes(schemaArray)

		for _, token := range templateFiles {
			bodyFilename := "body" + token
			rawBody, err := LoadTemplate(bodyFilename)
			if err != nil {
				continue
			}

			bakeCtx := StaticBakingContext{
				ServerName:         config.Hostname,
				Workgroup:          config.Workgroup,
				HostIP:             config.SystemIP.String(),
				DomainSuffix:       config.DomainName,
				InstanceID:         currentInstanceID,
				InstanceUUID:       currentInstanceUUID,
				MetadataVersion:    1,
				EnvelopeAttributes: envelopeAttrsStr,
				SchemaSlice:        schemaArray[:],
			}

			if token == "bye" {
				bakeCtx.MetadataVersion = 2
			}

			var actionURLBase string
			if token == "get" || token == "getresponse" {
				actionURLBase = versions.TransferSchema
			} else {
				actionURLBase = schemaArray[versions.Discovery]
			}

			var displayToken string
			switch token {
			case "probematches":
				displayToken = "ProbeMatches"
			case "resolvematches":
				displayToken = "ResolveMatches"
			case "hello":
				displayToken = "Hello"
			case "bye":
				displayToken = "Bye"
			case "getmetadata":
				displayToken = "GetMetadata"
			case "getresponse":
				displayToken = "GetResponse"
			default:

				displayToken = strings.ToTitle(token)
			}
			bakeCtx.ActionType = actionURLBase + "/" + displayToken

			bodyPreTmpl, err := template.New("body_pre_" + bodyFilename).Funcs(preFuncMap).Parse(rawBody)
			if err != nil {
				return fmt.Errorf("[%s|%s] Body pass-1 syntax parsing error: %w", schemaVersion, displayToken, err)
			}
			var bodyBuf bytes.Buffer
			if err := bodyPreTmpl.Execute(&bodyBuf, &bakeCtx); err != nil {
				return fmt.Errorf("[%s|%s] Body pass-1 execution failure: %w", schemaVersion, displayToken, err)
			}

			bakeCtx.BodyPayload = bodyBuf.String()

			headerPreTmpl, err := template.New("header_pre_" + bodyFilename).Funcs(preFuncMap).Parse(rawHeader)
			if err != nil {
				return fmt.Errorf("[%s|%s] Header pass-2 syntax parsing error: %w", schemaVersion, displayToken, err)
			}
			var mergedBuffer bytes.Buffer
			if err := headerPreTmpl.Execute(&mergedBuffer, &bakeCtx); err != nil {
				return fmt.Errorf("[%s|%s] Header pass-2 execution failure: %w", schemaVersion, displayToken, err)
			}

			finalTmpl, err := template.New(schemaVersion + "|" + displayToken).Funcs(runtimeFuncMap).Parse(mergedBuffer.String())
			if err != nil {
				return fmt.Errorf("[%s|%s] Final compilation generation matrix error: %w", schemaVersion, displayToken, err)
			}

			CombinedTemplateCache[schemaVersion][displayToken] = finalTmpl
		}
	}
	return nil
}
