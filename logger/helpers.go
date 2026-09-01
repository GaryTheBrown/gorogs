package logger

import (
	"strings"
	"time"
)

var (
	EnableColors bool = true
)

const (
	colorReset  string = "\033[0m"
	colorGreen  string = "\033[32m"
	colorYellow string = "\033[33m"
	colorRed    string = "\033[31m"
	colorCyan   string = "\033[36m"
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
		case "WARN":
			sb.WriteString(colorYellow)
		case "ERROR", "FATAL CRASH":
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
