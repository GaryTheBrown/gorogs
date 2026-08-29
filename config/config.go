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

type ConfigMap map[string]any

func (cm ConfigMap) GetSubSection(subSection string) ConfigMap {
	if gotValue, exists := cm[strings.ToLower(subSection)].(ConfigMap); exists {
		return gotValue
	}
	return nil
}

func (cm ConfigMap) Get[T any](key string, defaultValue T) T {
	if gotValue, exists := cm[strings.ToLower(key)].(T); exists {
		return gotValue
	}
	return defaultValue
}

func (cm ConfigMap) GetExists[T any](key string, defaultValue T) (T, bool) {
	if gotValue, exists := cm[strings.ToLower(key)].(T); exists {
		return gotValue, true
	}
	return defaultValue, false
}

func (cm ConfigMap) Exists(key string) bool {
	_, exists := cm[strings.ToLower(key)]
	return exists

}

func GetServiceConfig(key string) ConfigMap {
	if gotValue, exists := massConfigMap[strings.ToLower(key)].(ConfigMap); exists {
		return gotValue
	}
	return nil
}

func Get[T any](key string, defaultValue T) T {
	if gotValue, exists := massConfigMap[strings.ToLower(key)].(T); exists {
		return gotValue
	}
	return defaultValue
}

func GetExists[T any](key string, defaultValue T) (T, bool) {
	if gotValue, exists := massConfigMap[strings.ToLower(key)].(T); exists {
		return gotValue, true
	}
	return defaultValue, false
}

func Exists(key string) bool {
	_, exists := massConfigMap[strings.ToLower(key)]
	return exists

}
