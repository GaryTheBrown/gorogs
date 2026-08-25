package templates

import (
	"embed"
	"fmt"
	"io"
)

//go:embed xml/*.xml
var xmlFS embed.FS

func LoadTemplate(filename string) (string, error) {
	fullPath := "xml/" + filename + ".xml"

	file, err := xmlFS.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("compliance error: required asset '%s' is missing from binary map (run go generate)", fullPath)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
