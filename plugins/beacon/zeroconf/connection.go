package main

import (
	"context"
	"fmt"
	"gorogs/logger"
	"net"
	"os"
	"syscall"

	"golang.org/x/net/ipv4"
)

func (s *Struct) connectionStart() error {
	logger.Debug(Name, "[CONNECTION STARTING]")

	var err error
	s.multicastAddr, err = net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address block : %w", err)
	}

	lc := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) error {
			var opErr error
			c.Control(func(fd uintptr) {
				opErr = os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1))

				if opErr == nil {
					_ = os.NewSyscallError("setsockopt", syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, 0xf, 1))
				}
			})
			return opErr
		},
	}

	packetConn, err := lc.ListenPacket(context.Background(), "udp4", ":5353")
	if err != nil {
		return fmt.Errorf("failed to bind dual-mode mDNS socket port: %w", err)
	}
	s.conn = packetConn.(*net.UDPConn)

	p := ipv4.NewPacketConn(s.conn)
	if err = p.SetMulticastTTL(255); err != nil {
		return fmt.Errorf("failed to configure multicast TTL hop limit: %w", err)
	}
	_ = p.SetMulticastLoopback(true)

	ifaces, _ := net.Interfaces()
	joinedAny := false
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagMulticast != 0 {
			err = p.JoinGroup(&iface, &net.UDPAddr{IP: net.IPv4(224, 0, 0, 251)})
			if err == nil {
				joinedAny = true
			}
		}
	}

	if !joinedAny {
		return fmt.Errorf("failed to subscribe socket to mDNS multicast tracking group on any active network interface")
	}

	logger.Debug(Name, "[CONNECTION DONE]")
	return nil
}
