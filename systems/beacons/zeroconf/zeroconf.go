package zeroconf

import (
	"fmt"
	"net"
	"strings"

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
func (_ *Struct) Dependencies() []string                       { return nil }
func (_ *Struct) OrderAfter() []string                         { return []string{"nfs", "samba"} }
func (_ *Struct) Priority() int                                { return 100 }

func init() {
	systeminterface.Register(&Struct{})
}

func (s *Struct) Config(cm config.ConfigMap) {
	disabledStr := cm.Get("disabled", "")
	disabledSlice = strings.SplitSeq(strings.ToLower(disabledStr), ",")
	enabledStr := cm.Get("enabled", "")
	enabledSlice = strings.SplitSeq(strings.ToLower(enabledStr), ",")

	activebroadcaster = cm.Get("activebroadcaster", false)
	forceLocalDomainName = cm.Get("forcelocaldomainname", false)
	forceNoDomainName = cm.Get("forcelocaldomainname", false)
	singleTextRecord = cm.Get("singletextrecord", false)

	serverIcon = cm.Get("servericon", "nas")
	serverName = cm.Get("servername", config.Hostname)

}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")
	if forceLocalDomainName {
		hostParam = fmt.Sprintf("%s.local", serverName)
		logger.Debug(Name, "[FORCED LOCAL MODE]")
	} else if forceNoDomainName {
		hostParam = fmt.Sprintf("%s", serverName)
		logger.Debug(Name, "[FORCED NOLOCAL MODE]")

	} else if config.DomainName == "" {
		hostParam = fmt.Sprintf("%s.local", serverName)
		logger.Debug(Name, "[LOCAL MODE]")
	} else {
		hostParam = fmt.Sprintf("%s.%s", serverName, config.DomainName)
		logger.Debug(Name, "[FQDN MODE]")
	}
	tcpLocal = "_tcp.local."
	udpLocal = "_udp.local."
	hostTarget = fmt.Sprintf("%s.", hostParam)
	servicesMetaRecord = fmt.Sprintf("_services._dns-sd.%s", udpLocal)

	AddrSetup()

	s.sState = systeminterface.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")
	if err := s.connectionStart(); err != nil {
		return err
	}

	s.done = make(chan struct{})

	go s.listenForQueries()
	logger.Debug(Name, "[STARTED LISTNER]")

	s.broadcastHello()
	logger.Debug(Name, "[BROADCAST HELLO]")

	if activebroadcaster {
		go s.activeBroadcaster()
		logger.Debug(Name, "[STARTED REFRESH TICKER]")
	}
	s.sState = systeminterface.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.Debug(Name, "Stopping ZeroCONF...")

	if s.done != nil {
		close(s.done)
	}

	if s.conn != nil {
		s.BroadcastBye()
		logger.Debug(Name, "[BROADCAST BYE]")

		_ = s.conn.Close()
		s.conn = nil
		logger.Debug(Name, "[CLOSE LISTNER]")
	}

	s.sState = systeminterface.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.conn == nil {
		return fmt.Errorf("ZeroCONF is not initialized")
	}
	return nil
}
