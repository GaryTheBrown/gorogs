package zeroconf

import (
	"fmt"
	"net"
	"strings"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "ZeroCONF"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

var (
	localHostTarget    = fmt.Sprintf("%s.local.", strings.ToLower(config.Hostname))
	fqdnHostTarget     = fmt.Sprintf("%s.%s.", strings.ToLower(config.Hostname), strings.ToLower(config.DomainName))
	servicesMetaRecord = "_services._dns-sd._udp.local."
	txtRecords         = []string{"path=/", fmt.Sprintf("host=%s", fqdnHostTarget)}
	nfsAddr            = "_nfs._tcp.local."
	smbAddr            = "_smb._tcp.local."
)

type Struct struct {
	sState        systeminterface.SysStateEnum
	conn          *net.UDPConn
	multicastAddr *net.UDPAddr
	done          chan struct{}
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Config() {
	// NOTHING TO CONFIGURE IN HERE
}

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")
	var err error
	s.multicastAddr, err = net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		logger.FatalF(Name, "failed to resolve multicast address block: %w", err)
	}
	logger.DebugAppend(Name, "[RESOLVE MULTICAST ADDRESS]")

	s.conn, err = net.ListenMulticastUDP("udp4", nil, s.multicastAddr)
	if err != nil {
		logger.FatalF(Name, "failed to join kernel ZeroCONF multicast loop: %w", err)
	}
	logger.DebugAppend(Name, "[LISTEN TO MULTICAST ADDRESS]")

	s.done = make(chan struct{})
	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")

	s.broadcastAnnouncement(120)
	logger.DebugAppend(Name, "[BROADCAST HELLO]")

	go s.listenForQueries()
	logger.DebugAppend(Name, "[STARTED LISTNER]")

	s.sState = systeminterface.STARTED
	logger.DebugEnd(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.DebugContinue(Name, "Stopping ZeroCONF daemon threads...")

	if s.done != nil {
		close(s.done)
	}

	if s.conn != nil {
		s.broadcastAnnouncement(0)
		logger.DebugAppend(Name, "[BROADCAST BYE]")
		time.Sleep(500 * time.Millisecond)

		_ = s.conn.Close()
		s.conn = nil
		logger.DebugAppend(Name, "[CLOSE LISTNER]")
	}

	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.conn == nil {
		return fmt.Errorf("ZeroCONF is not initialized")
	}
	return nil
}
