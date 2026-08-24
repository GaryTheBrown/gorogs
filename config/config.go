package config

import (
	"net"
	"strings"
)

const (
	ConstOriginalShareRoot = "/srv"
)

var (
	Hostname   string
	DomainName string
	SystemIP   net.IP

	ShareRoot = ConstOriginalShareRoot
	Workgroup = "WORKGROUP"
)

func IsDisabled(service string) bool {
	_, exists := disabled[strings.ToLower(service)]
	return exists
}

func IsEnabled(service string) bool {
	_, exists := disabled[strings.ToLower(service)]
	return exists
}

func GetServiceConfig(service string) map[string]any {
	if gotValue, exists := massConfigMap[service]; exists {
		if subMap, typeRight := gotValue.(map[string]any); typeRight {
			return subMap
		}
	}
	return nil
}

func GetSingleServiceConfig(service string) any {
	if gotValue, exists := massConfigMap[service]; exists {
		if _, isMap := gotValue.(map[string]any); !isMap {
			return gotValue
		}
	}
	return nil
}

func GetSingleServiceConfigString(service string, defaultString string) string {
	if gotValue, exists := massConfigMap[service].(string); exists {
		return gotValue
	}
	return defaultString
}

func GetSingleServiceConfigBool(service string, defaultBool bool) bool {
	if gotValue, exists := massConfigMap[service].(bool); exists {
		return gotValue
	}
	return defaultBool
}

func GetSingleServiceConfigInt(service string, defaultInt int) int {
	if gotValue, exists := massConfigMap[service].(int); exists {
		return gotValue
	}
	return defaultInt
}

func GetSingleServiceConfigFloat(service string, defaultFloat float64) float64 {
	if gotValue, exists := massConfigMap[service].(float64); exists {
		return gotValue
	}
	return defaultFloat
}
