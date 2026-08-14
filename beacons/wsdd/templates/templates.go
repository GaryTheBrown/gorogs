package templates

import (
	"embed"
	"fmt"
	"gorogs/logger"
	"io"
)

//go:embed xml/*.xml
var xmlFS embed.FS

func LoadTemplate(filename string) (string, error) {
	fullPath := "xml/" + filename + ".xml"
	logger.Debug("wsdd", fmt.Sprintf("Accessing binary embedded template memory maps for asset path: %s", fullPath))

	file, err := xmlFS.Open(fullPath)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Asset retrieval error: asset target path '%s' is missing or corrupted inside memory space", fullPath), err)
		return "", fmt.Errorf("compliance error: required asset '%s' is missing from binary map (run go generate)", fullPath)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		logger.Error("wsdd", fmt.Sprintf("Buffer mapping read failure executing dynamic read buffer extraction stream on file target: %s", fullPath), err)
		return "", err
	}

	logger.Debug("wsdd", fmt.Sprintf("Asset allocation pass successfully completed. Ingested %d text bytes from embedded file layer: %s", len(data), fullPath))
	return string(data), nil
}
