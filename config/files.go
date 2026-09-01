package config

import (
	"errors"
	"fmt"
	"os"
)

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func init() {
	if fileExists("/etc/hosts") {
		return
	}
	hostsContent := fmt.Sprintf(
		"127.0.0.1\tlocalhost\n"+
			"::1\tlocalhost ip6-localhost ip6-loopback\n"+
			"%s\t%s\n",
		SystemIP.String(), Hostname,
	)
	os.WriteFile("/etc/hosts", []byte(hostsContent), 0644)

	if fileExists("/etc/nsswitch.conf") {
		return
	}
	nsswitchContent := "passwd:         files\n" +
		"group:          files\n" +
		"shadow:         files\n" +
		"gshadow:        files\n\n" +
		"hosts:          files dns\n" +
		"networks:       files\n\n" +
		"protocols:      db files\n" +
		"services:       db files\n" +
		"ethers:         db files\n" +
		"rpc:            db files\n"

	os.WriteFile("/etc/nsswitch.conf", []byte(nsswitchContent), 0644)
}
