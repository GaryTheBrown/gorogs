package main

import (
	"fmt"
	"gorogs/config"
	"os"
	"strings"
)

type ServiceDefinition struct {
	Name       string
	Label      string
	Filename   string
	Type       string
	Port       int
	TxtRecords map[string]string
}

var services []ServiceDefinition

func (sd *ServiceDefinition) ServiceBytes() []byte {
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

func (sd *ServiceDefinition) FilePath() string {
	return fmt.Sprintf("%s/%s.service", serviceConfigPath, sd.Filename)
}
func (sd *ServiceDefinition) WriteService() error {
	_, sambaDisabled := disabled[sd.Name]
	if !sambaDisabled && !config.IsDisabled(sd.Name) {
		return os.WriteFile(sd.FilePath(), sd.ServiceBytes(), 0644)
	}
	return nil
}

func init() {
	services = []ServiceDefinition{
		{
			Name:     "device-Info",
			Label:    "Device Info",
			Filename: "deviceinfo",
			Type:     "_device-info._tcp",
			Port:     445,
			TxtRecords: map[string]string{
				"model": "TimeCapsule8,119",
			},
		},
		{
			Name:     "samba",
			Label:    "Samba",
			Filename: "smb",
			Type:     "_smb._tcp",
			Port:     445,
		},
		{
			Name:     "nfs",
			Label:    "NFS",
			Filename: "nfs",
			Type:     "_nfs._tcp",
			Port:     2049,
		},
	}
}
