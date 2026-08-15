package logger

import (
	"strings"
	"time"
)

var EnableColors = true

const (
	colorReset  = "\033[0m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorCyan   = "\033[36m"
)

func formatPrefix(subSystem, messageType string) string {
	ts := time.Now().Format("2006-01-02 15:04:05")
	mt := strings.ToUpper(messageType)

	var sb strings.Builder
	sb.WriteString("[")
	sb.WriteString(ts)
	sb.WriteString("]")
	sb.WriteString(" ")

	if EnableColors {
		switch mt {
		case "INFO":
			sb.WriteString(colorGreen)
		case "ERROR":
			sb.WriteString(colorYellow)
		case "FATAL CRASH":
			sb.WriteString(colorRed)
		case "DEBUG":
			sb.WriteString(colorCyan)
		}
	}

	sb.WriteString("[")
	sb.WriteString(strings.ToUpper(messageType))
	sb.WriteString("]")

	if EnableColors {
		sb.WriteString(colorReset)
	}

	sb.WriteString(" ")

	if subSystem != "" {
		parts := strings.SplitSeq(strings.ToUpper(subSystem), ".")
		for part := range parts {
			if part != "" {
				sb.WriteString("[")
				sb.WriteString(part)
				sb.WriteString("]")
			}
		}
		sb.WriteString(" ")
	}

	return sb.String()
}
