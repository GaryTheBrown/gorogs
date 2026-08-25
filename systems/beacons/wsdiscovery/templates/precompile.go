package templates

import (
	"bytes"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/versions"

	"io/fs"
	"strings"
	"text/template"
)

func PreCompileTemplates() {
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
		logger.Fatal(Name, "Fatal initialization break: Missing core envelope header asset template", err)
	}

	entries, err := fs.ReadDir(xmlFS, "xml")
	if err != nil {
		logger.Fatal(Name, "Failed to scan dynamic template tokens from embedFS storage layer", err)
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
				logger.FatalF(Name, "[%s|%s] Body pass-1 syntax parsing error", err, schemaVersion, displayToken)
			}
			var bodyBuf bytes.Buffer
			if err := bodyPreTmpl.Execute(&bodyBuf, &bakeCtx); err != nil {
				logger.FatalF(Name, "[%s|%s] Body pass-1 execution failure", err, schemaVersion, displayToken)
			}

			bakeCtx.BodyPayload = bodyBuf.String()

			headerPreTmpl, err := template.New("header_pre_" + bodyFilename).Funcs(preFuncMap).Parse(rawHeader)
			if err != nil {
				logger.FatalF(Name, "[%s|%s] Header pass-2 syntax parsing error", err, schemaVersion, displayToken)
			}
			var mergedBuffer bytes.Buffer
			if err := headerPreTmpl.Execute(&mergedBuffer, &bakeCtx); err != nil {
				logger.FatalF(Name, "[%s|%s] Header pass-2 execution failure", err, schemaVersion, displayToken)
			}

			finalTmpl, err := template.New(schemaVersion + "|" + displayToken).Funcs(runtimeFuncMap).Parse(mergedBuffer.String())
			if err != nil {
				logger.FatalF(Name, "[%s|%s] Final compilation generation matrix error", err, schemaVersion, displayToken)
			}

			CombinedTemplateCache[schemaVersion][displayToken] = finalTmpl
		}
	}
}
