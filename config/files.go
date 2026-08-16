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

func init() { //_etc_hosts() {
	if !fileExists("/etc/hosts") {
		hostsContent := fmt.Sprintf(
			"127.0.0.1\tlocalhost\n"+
				"::1\tlocalhost ip6-localhost ip6-loopback\n"+
				"%s\t%s\n",
			SystemIP.String(), Hostname,
		)
		if err := os.WriteFile("/etc/hosts", []byte(hostsContent), 0644); err != nil {
			// return fmt.Errorf("failed to rewrite system hosts table: %w", err)
		}
	}
}
func init() { //_etc_nsswitch_conf() {
	if !fileExists("/etc/nsswitch.conf") {
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

		if err := os.WriteFile("/etc/nsswitch.conf", []byte(nsswitchContent), 0644); err != nil {
			// return fmt.Errorf("failed to rewrite system name service switch table: %w", err)
		}
	}
}
