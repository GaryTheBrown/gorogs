package main

import (
	"fmt"
	"strings"
)

type ServiceDefinition struct {
	Name       string
	Type       string
	Port       int
	TxtRecords map[string]string
}

var services map[string]ServiceDefinition

func (sd *ServiceDefinition) Service() []byte {
	var extra strings.Builder
	for key, value := range sd.TxtRecords {
		fmt.Fprintf(&extra, "<txt-record>%s=%s</txt-record>", key, value)
	}

	rawStr := `<?xml version="1.0" standalone='no'?><!--*-nxml-*-->
<!DOCTYPE service-group SYSTEM "avahi-service.dtd">
<service-group>
  <name replace-wildcards="yes">%%h</name>
  <service><type>%s</type><port>%d</port>%s</service>
</service-group>`
	xmlString := fmt.Sprintf(rawStr, sd.Type, sd.Port, extra.String())

	return []byte(xmlString)
}

func init() {
	services = map[string]ServiceDefinition{
		"device-info": {
			Name: "Device Info",
			Type: "_device-info._tcp",
			Port: 445,
			TxtRecords: map[string]string{
				"model": "TimeCapsule8,119",
			},
		},
		"samba": {
			Name: "Samba",
			Type: "_smb._tcp",
			Port: 445,
		},
		"nfs": {
			Name: "NFS",
			Type: "_nfs._tcp",
			Port: 2049,
			// TxtRecords: map[string]string{
			// 	"path": "/",
			// },
		},
	}
}
