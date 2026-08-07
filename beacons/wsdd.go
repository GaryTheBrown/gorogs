package beacons

import (
	"crypto/rand"
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
	"syscall"

	"gorogs/config"
	"gorogs/logger"
)

type WsddBeacon struct {
	conn       *net.UDPConn
	running    bool
	waitGroup  sync.WaitGroup
	instanceID string
	multicast  *net.UDPAddr
}

const (
	wsdActionProbeMatch = "ProbeMatches"
	wsdActionHello      = "Hello"
	wsdActionBye        = "Bye"

	wsdEnvelopeTemplate = `<?xml version="1.0" encoding="utf-8"?>` +
		`<s:Envelope xmlns:s="http://w3.org" xmlns:a="%s" xmlns:d="%s" xmlns:pub="http://microsoft.com">` +
		`<s:Header>` +
		`<a:Action>%s/%s</a:Action>` +
		`<a:MessageID>urn:uuid:%s</a:MessageID>` +
		`%s` +
		`<a:To>%s/role/anonymous</a:To>` +
		`</s:Header>` +
		`<s:Body>%s</s:Body>` +
		`</s:Envelope>`

	wsdProbeMatchBody = `<d:ProbeMatches><d:ProbeMatch>` +
		`<a:EndpointReference><a:Address>urn:uuid:%s</a:Address></a:EndpointReference>` +
		`<d:Types>pub:Computer</d:Types><d:Scopes /><d:XAddrs>%s:5357</d:XAddrs><d:MetadataVersion>1</d:MetadataVersion>` +
		`</d:ProbeMatch></d:ProbeMatches>`

	wsdHelloBody = `<d:Hello>` +
		`<a:EndpointReference><a:Address>urn:uuid:%s</a:Address></a:EndpointReference>` +
		`<d:Types>pub:Computer</d:Types><d:XAddrs>%s:5357</d:XAddrs><d:MetadataVersion>1</d:MetadataVersion>` +
		`</d:Hello>`

	wsdByeBody = `<d:Bye>` +
		`<a:EndpointReference><a:Address>urn:uuid:%s</a:Address></a:EndpointReference>` +
		`</d:Bye>`
)

func (w *WsddBeacon) Setup() error {
	logger.Info("WSDD", "Evaluating Windows WS-Discovery (WSDD) service prerequisites...")

	if !config.Instance.SambaEnabled {
		logger.Info("WSDD", "Samba storage daemon is disabled. Bypassing dependent WSDD subsystem.")
		return ErrServiceDisabled
	}
	if !config.Instance.WsddEnabled {
		logger.Info("WSDD", "WSDD engine is explicitly disabled via configuration switches. Bypassing subsystem.")
		return ErrServiceDisabled
	}

	uuidBytes := make([]byte, 16)
	if _, err := rand.Read(uuidBytes); err != nil {
		return fmt.Errorf("failed to generate secure system instance token: %w", err)
	}
	w.instanceID = fmt.Sprintf("%x-%x-%x-%x-%x", uuidBytes[0:4], uuidBytes[4:6], uuidBytes[6:8], uuidBytes[8:10], uuidBytes[10:])
	w.multicast = &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 3702}

	logger.Info("WSDD", "Pre-flight validation complete. Component ready for boot operations.")
	return nil
}

func (w *WsddBeacon) Start() error {
	logger.Info("WSDD", "Assembling raw multi-namespace network sockets...")

	addr := &net.UDPAddr{IP: net.IPv4zero, Port: 3702}
	s, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to establish local UDP port 3702 socket binding: %w", err)
	}
	w.conn = s

	file, err := s.File()
	if err != nil {
		s.Close()
		return fmt.Errorf("failed to extract socket file descriptors: %w", err)
	}
	defer file.Close()

	if err := syscall.SetsockoptInt(int(file.Fd()), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
		s.Close()
		return fmt.Errorf("failed to configure SO_REUSEADDR socket layers: %w", err)
	}

	mreq := &syscall.IPMreq{
		Multiaddr: [4]byte{239, 255, 255, 250},
		Interface: [4]byte{0, 0, 0, 0},
	}

	if err := syscall.SetsockoptIPMreq(int(file.Fd()), syscall.IPPROTO_IP, syscall.IP_ADD_MEMBERSHIP, mreq); err != nil {
		s.Close()
		return fmt.Errorf("failed to submit IGMP multicast channel subscription maps: %w", err)
	}

	w.running = true
	w.waitGroup.Add(1)

	w.sendAutonomousAnnouncement(wsdActionHello)
	go w.processIncomingTraffic()

	logger.Info("WSDD", "WSDD Service Active: Listening and broadcasting on 239.255.255.250:3702")
	return nil
}

func (w *WsddBeacon) processIncomingTraffic() {
	defer w.waitGroup.Done()
	buffer := make([]byte, 4096)

	reAddr := regexp.MustCompile(`xmlns:(?:[a-zA-Z0-9_\-]*?)="(http://schemas\.xmlsoap\.org/ws/[0-9]{4}/[0-9]{2}/addressing)"`)
	reDisc := regexp.MustCompile(`xmlns:(?:[a-zA-Z0-9_\-]*?)="(http://schemas\.xmlsoap\.org/ws/[0-9]{4}/[0-9]{2}/discovery)"`)
	reAction := regexp.MustCompile(`<(?:[a-zA-Z0-9_\-]*?:)?Action>(.*?)/Probe</(?:[a-zA-Z0-9_\-]*?:)?Action>`)
	reMsgID := regexp.MustCompile(`<(?:[a-zA-Z0-9_\-]*:)?MessageID>(?:urn:uuid:)?([0-9a-fA-F\-]+)</(?:[a-zA-Z0-9_\-]*:)?MessageID>`)

	for w.running {
		n, srcAddr, err := w.conn.ReadFromUDP(buffer)
		if err != nil {
			if !w.running {
				return
			}
			continue
		}

		payload := string(buffer[:n])

		if strings.Contains(payload, "Probe") {
			logger.Debug("WSDD", fmt.Sprintf("Intercepted Probe envelope from client %s:\n%s", srcAddr.String(), strings.TrimSpace(payload)))

			addrMatch := reAddr.FindStringSubmatch(payload)
			nsAddressing := "http://xmlsoap.org"
			if len(addrMatch) > 1 {
				nsAddressing = addrMatch[1]
			}

			discMatch := reDisc.FindStringSubmatch(payload)
			nsDiscovery := "http://xmlsoap.org"
			if len(discMatch) > 1 {
				nsDiscovery = discMatch[1]
			}

			actionMatch := reAction.FindStringSubmatch(payload)
			baseAction := nsDiscovery
			if len(actionMatch) > 1 {
				baseAction = actionMatch[1]
			}

			msgIDMatch := reMsgID.FindStringSubmatch(payload)
			if len(msgIDMatch) < 2 {
				continue
			}
			clientMsgID := msgIDMatch[1]

			replyMsgUUID := w.generateRandomUUID()
			relatesToBlock := fmt.Sprintf("<a:RelatesTo>urn:uuid:%s</a:RelatesTo>", clientMsgID)
			bodyContent := fmt.Sprintf(wsdProbeMatchBody, w.instanceID, config.Instance.ContainerIP.String())

			replyXML := fmt.Sprintf(wsdEnvelopeTemplate,
				nsAddressing,
				nsDiscovery,
				baseAction,
				wsdActionProbeMatch,
				replyMsgUUID,
				relatesToBlock,
				nsAddressing,
				bodyContent,
			)

			logger.Debug("WSDD", fmt.Sprintf("Transmitting mirrored ProbeMatch XML stream back to client %s:\n%s", srcAddr.String(), replyXML))
			_, _ = w.conn.WriteToUDP([]byte(replyXML), srcAddr)
		}
	}
}

func (w *WsddBeacon) sendAutonomousAnnouncement(actionType string) {
	if w.conn == nil {
		return
	}

	nsAddressing := "http://xmlsoap.org"
	nsDiscovery := "http://xmlsoap.org"

	msgUUID := w.generateRandomUUID()
	var bodyContent string

	if actionType == wsdActionHello {
		bodyContent = fmt.Sprintf(wsdHelloBody, w.instanceID, config.Instance.ContainerIP.String())
	} else {
		bodyContent = fmt.Sprintf(wsdByeBody, w.instanceID)
	}

	envelopeXML := fmt.Sprintf(wsdEnvelopeTemplate,
		nsAddressing,
		nsDiscovery,
		nsDiscovery,
		actionType,
		msgUUID,
		"",
		nsAddressing,
		bodyContent,
	)

	logger.Debug("WSDD", fmt.Sprintf("Broadcasting autonomous network advertisement event [%s] onto wire:\n%s", actionType, envelopeXML))
	_, _ = w.conn.WriteToUDP([]byte(envelopeXML), w.multicast)
}

func (w *WsddBeacon) generateRandomUUID() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return fmt.Sprintf("%x-%x-%x-%x-%x", bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:])
}

func (w *WsddBeacon) Healthcheck() error {
	if !w.running || w.conn == nil {
		return fmt.Errorf("wsdd advertisement background routine has collapsed or closed")
	}
	return nil
}

func (w *WsddBeacon) IsCritical() bool { return false }

func (w *WsddBeacon) Stop() error {
	if !w.running {
		return nil
	}
	w.running = false
	w.sendAutonomousAnnouncement(wsdActionBye)
	if w.conn != nil {
		w.conn.Close()
	}
	w.waitGroup.Wait()
	return nil
}
