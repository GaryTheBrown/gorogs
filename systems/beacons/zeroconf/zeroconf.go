package zeroconf

import (
	"fmt"
	"net"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/systeminterface"

	"github.com/miekg/dns"
)

const (
	Name       = "ZeroCONF"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

type Struct struct {
	sState systeminterface.SysStateEnum
	conn   *net.UDPConn
	done   chan struct{}
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	// logger.Info(m.Name(), "Evaluating network discovery broadcast requirements...")
	s.sState = systeminterface.SETUP
	// logger.Info(m.Name(), "Pre-flight checks passed. Service registration profile is valid.")
}

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Initializing proactive proxy-routing mDNS discovery engine...")

	multicastAddr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address block: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	if err != nil {
		return fmt.Errorf("failed to join kernel mDNS multicast loop: %w", err)
	}
	s.conn = conn
	s.done = make(chan struct{})

	s.broadcastAnnouncement(120)

	go s.listenForQueries()

	logger.Info(s.Name(), "Universal mDNS service discovery proxies active and broadcasting Hello packets.")
	s.sState = systeminterface.STARTED
	return nil
}

func (s *Struct) Stop() {
	logger.Info(s.Name(), "Initiating shutdown sequence on unified mDNS channels...")

	if s.done != nil {
		close(s.done)
	}

	if s.conn != nil {
		s.broadcastAnnouncement(0)
		time.Sleep(500 * time.Millisecond)

		_ = s.conn.Close()
		s.conn = nil
	}

	logger.Info(s.Name(), "mDNS broadcast beacons dropped cleanly from network space.")
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.conn == nil {
		return fmt.Errorf("unified mDNS discovery server connection instance is uninitialized")
	}
	return nil
}
func (s *Struct) listenForQueries() {
	localHostTarget := fmt.Sprintf("%s.local.", config.Hostname)
	fqdnHostTarget := fmt.Sprintf("%s.%s.", config.Hostname, config.DomainName)

	servicesMetaRecord := "_services._dns-sd._udp.local."
	txtRecords := []string{"path=/", fmt.Sprintf("host=%s", fqdnHostTarget)}

	buf := make([]byte, 1500)
	for {
		select {
		case <-s.done:
			return
		default:
			n, remoteAddr, err := s.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			msg := new(dns.Msg)
			if err := msg.Unpack(buf[:n]); err != nil {
				continue
			}

			for _, q := range msg.Question {
				logger.DebugF(s.Name(), "Incoming Query from %s: Name=%s Type=%s", remoteAddr.String(), q.Name, dns.TypeToString[q.Qtype])

				if q.Name == localHostTarget || q.Name == fqdnHostTarget || q.Name == servicesMetaRecord || q.Name == "_nfs._tcp.local." || q.Name == "_smb._tcp.local." {
					resp := new(dns.Msg)
					resp.SetReply(msg)
					resp.Compress = true
					resp.Authoritative = true

					resp.Answer = append(resp.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: localHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
						A:   config.SystemIP,
					})
					resp.Answer = append(resp.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: fqdnHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
						A:   config.SystemIP,
					})

					// Handle DNS-SD Service Enumeration (Browsing for available service types)
					if q.Name == servicesMetaRecord {
						if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") {
							resp.Answer = append(resp.Answer, &dns.PTR{
								Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
								Ptr: "_nfs._tcp.local.",
							})
						}
						if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") {
							resp.Answer = append(resp.Answer, &dns.PTR{
								Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
								Ptr: "_smb._tcp.local.",
							})
						}
					}

					if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") && q.Name == "_nfs._tcp.local." {
						ptrName := "_nfs._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: instanceName,
						})
						resp.Extra = append(resp.Extra, &dns.SRV{
							Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 120},
							Priority: 0, Weight: 0, Port: 2049, Target: fqdnHostTarget,
						})
						for _, txt := range txtRecords {
							resp.Extra = append(resp.Extra, &dns.TXT{
								Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 120},
								Txt: []string{txt},
							})
						}
					}

					if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") && q.Name == "_smb._tcp.local." {
						ptrName := "_smb._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: instanceName,
						})
						resp.Extra = append(resp.Extra, &dns.SRV{
							Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 120},
							Priority: 0, Weight: 0, Port: 445, Target: fqdnHostTarget,
						})
						for _, txt := range txtRecords {
							resp.Extra = append(resp.Extra, &dns.TXT{
								Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 120},
								Txt: []string{txt},
							})
						}
					}

					out, err := resp.Pack()
					if err == nil {
						logger.DebugF(s.Name(), "Sending Targeted Response to %s for %s", remoteAddr.String(), q.Name)
						_, _ = s.conn.WriteToUDP(out, remoteAddr)
					}
				}
			}
		}
	}
}

func (s *Struct) broadcastAnnouncement(ttl uint32) {
	localHostTarget := fmt.Sprintf("%s.local.", config.Hostname)
	fqdnHostTarget := fmt.Sprintf("%s.%s.", config.Hostname, config.DomainName)

	multicastAddr, _ := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")

	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Compress = true

	msg.Answer = append(msg.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: localHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   config.SystemIP,
	})
	msg.Answer = append(msg.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: fqdnHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   config.SystemIP,
	})

	txtRecords := []string{
		"path=/",
		fmt.Sprintf("host=%s", fqdnHostTarget),
	}

	servicesMetaRecord := "_services._dns-sd._udp.local."

	if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") {
		ptrName := "_nfs._tcp.local."
		instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: ptrName,
		})
		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: instanceName,
		})

		msg.Extra = append(msg.Extra, &dns.SRV{
			Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
			Priority: 0, Weight: 0, Port: 2049, Target: fqdnHostTarget,
		})

		for _, txt := range txtRecords {
			msg.Extra = append(msg.Extra, &dns.TXT{
				Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
				Txt: []string{txt},
			})
		}
	}

	if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") {
		ptrName := "_smb._tcp.local."
		instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: ptrName,
		})
		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: instanceName,
		})

		msg.Extra = append(msg.Extra, &dns.SRV{
			Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
			Priority: 0, Weight: 0, Port: 445, Target: fqdnHostTarget,
		})

		for _, txt := range txtRecords {
			msg.Extra = append(msg.Extra, &dns.TXT{
				Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
				Txt: []string{txt},
			})
		}
	}

	out, err := msg.Pack()
	if err == nil {
		logger.DebugF(s.Name(), "Broadcasting Proactive Announcement Packet (Answers: %d, Extras: %d, TTL: %d)", len(msg.Answer), len(msg.Extra), ttl)

		_, _ = s.conn.WriteToUDP(out, multicastAddr)
		time.Sleep(100 * time.Millisecond)
		_, _ = s.conn.WriteToUDP(out, multicastAddr)
	}
}
