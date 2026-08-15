package utils

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strings"

	"gorogs/logger"
)

func QueryNetworkLayout() (net.IP, net.IP, error) {
	containerIP := net.ParseIP("127.0.0.1")
	gatewayIP := net.ParseIP("127.0.0.1")

	conn, err := net.Dial("udp", "10.255.255.255:1")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to evaluate internal network socket interface mapping: %w", err)
	}
	defer conn.Close()

	localAddr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, nil, fmt.Errorf("failed to type-assert local network address space")
	}
	containerIP = localAddr.IP

	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open kernel network routing definitions: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		if fields[1] == "00000000" {
			gwHex := fields[2]

			if len(gwHex) != 8 {
				logger.DebugF("NETUTIL", "Skipping parsed route entry: hex length (%d) != 8 characters.", len(gwHex))
				continue
			}

			gwBytes, err := hex.DecodeString(gwHex)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse gateway hex token: %w", err)
			}

			gatewayIP = net.IPv4(gwBytes[3], gwBytes[2], gwBytes[1], gwBytes[0])
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("error reading routing tables data stream: %w", err)
	}

	return containerIP, gatewayIP, nil
}

func InitializeNetworkEnvironment(containerIP net.IP) error {
	logger.Info("NETUTIL", "Synchronizing local system file structures with network layout...")

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}

	hostsContent := fmt.Sprintf(
		"127.0.0.1\tlocalhost\n"+
			"::1\tlocalhost ip6-localhost ip6-loopback\n"+
			"%s\t%s\n",
		containerIP.String(), hostname,
	)

	if err := os.WriteFile("/etc/hosts", []byte(hostsContent), 0644); err != nil {
		return fmt.Errorf("failed to rewrite system hosts table: %w", err)
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

	if err := os.WriteFile("/etc/nsswitch.conf", []byte(nsswitchContent), 0644); err != nil {
		return fmt.Errorf("failed to rewrite system name service switch table: %w", err)
	}

	logger.Info("NETUTIL", "System network files successfully initialized and synchronized on disk.")
	return nil
}
