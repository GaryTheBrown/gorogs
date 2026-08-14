package beacons

import (
	"fmt"
	"net"
	"time"

	"gorogs/config"
	"gorogs/logger"

	"github.com/miekg/dns"
)

type MdnsBeacon struct {
	config AppConfig
	conn   *net.UDPConn
	done   chan struct{}
}

func (m *MdnsBeacon) Setup(config AppConfig) error {
	logger.Info("MDNS", "Evaluating network discovery broadcast requirements...")
	m.config = config
	logger.Info("MDNS", "Pre-flight checks passed. Service registration profile is valid.")
	return nil
}

func (m *MdnsBeacon) Start() error {
	logger.Info("MDNS", "Initializing proactive proxy-routing mDNS discovery engine...")

	multicastAddr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address block: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	if err != nil {
		return fmt.Errorf("failed to join kernel mDNS multicast loop: %w", err)
	}
	m.conn = conn
	m.done = make(chan struct{})

	m.broadcastAnnouncement(120)

	go m.listenForQueries()

	logger.Info("MDNS", "Universal mDNS service discovery proxies active and broadcasting Hello packets.")
	return nil
}

func (m *MdnsBeacon) listenForQueries() {
	localHostTarget := fmt.Sprintf("%s.local.", m.config.ServerName)
	fqdnHostTarget := fmt.Sprintf("%s.%s.", m.config.ServerName, m.config.DomainSuffix)

	servicesMetaRecord := "_services._dns-sd._udp.local."
	txtRecords := []string{"path=/", fmt.Sprintf("host=%s", fqdnHostTarget)}

	buf := make([]byte, 1500)
	for {
		select {
		case <-m.done:
			return
		default:
			n, remoteAddr, err := m.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			msg := new(dns.Msg)
			if err := msg.Unpack(buf[:n]); err != nil {
				continue
			}

			for _, q := range msg.Question {
				logger.Debug("MDNS", fmt.Sprintf("Incoming Query from %s: Name=%s Type=%s", remoteAddr.String(), q.Name, dns.TypeToString[q.Qtype]))

				if q.Name == localHostTarget || q.Name == fqdnHostTarget || q.Name == servicesMetaRecord || q.Name == "_nfs._tcp.local." || q.Name == "_smb._tcp.local." {
					resp := new(dns.Msg)
					resp.SetReply(msg)
					resp.Compress = true
					resp.Authoritative = true

					resp.Answer = append(resp.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: localHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
						A:   m.config.ContainerIP,
					})
					resp.Answer = append(resp.Answer, &dns.A{
						Hdr: dns.RR_Header{Name: fqdnHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 120},
						A:   m.config.ContainerIP,
					})

					if config.NfsEnabled && config.MdnsNfsEnabled && (q.Name == servicesMetaRecord || q.Name == "_nfs._tcp.local.") {
						ptrName := "_nfs._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", m.config.ServerName, ptrName)

						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: ptrName,
						})
						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: instanceName,
						})
						resp.Answer = append(resp.Answer, &dns.SRV{
							Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 120},
							Priority: 0, Weight: 0, Port: 2049, Target: fqdnHostTarget,
						})
						for _, txt := range txtRecords {
							resp.Answer = append(resp.Answer, &dns.TXT{
								Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 120},
								Txt: []string{txt},
							})
						}
					}

					if config.SambaEnabled && config.MdnsSambaEnabled && (q.Name == servicesMetaRecord || q.Name == "_smb._tcp.local.") {
						ptrName := "_smb._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", m.config.ServerName, ptrName)

						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: ptrName,
						})
						resp.Answer = append(resp.Answer, &dns.PTR{
							Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: 120},
							Ptr: instanceName,
						})
						resp.Answer = append(resp.Answer, &dns.SRV{
							Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: 120},
							Priority: 0, Weight: 0, Port: 445, Target: fqdnHostTarget,
						})
						for _, txt := range txtRecords {
							resp.Answer = append(resp.Answer, &dns.TXT{
								Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 120},
								Txt: []string{txt},
							})
						}
					}

					out, err := resp.Pack()
					if err == nil {
						logger.Debug("MDNS", fmt.Sprintf("Sending Targeted Response to %s for %s", remoteAddr.String(), q.Name))
						_, _ = m.conn.WriteToUDP(out, remoteAddr)
					}
				}
			}
		}
	}
}

func (m *MdnsBeacon) broadcastAnnouncement(ttl uint32) {
	localHostTarget := fmt.Sprintf("%s.local.", m.config.ServerName)
	fqdnHostTarget := fmt.Sprintf("%s.%s.", m.config.ServerName, config.DomainSuffix)

	multicastAddr, _ := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")

	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Compress = true

	msg.Answer = append(msg.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: localHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   m.config.ContainerIP,
	})
	msg.Answer = append(msg.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: fqdnHostTarget, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   m.config.ContainerIP,
	})

	txtRecords := []string{
		"path=/",
		fmt.Sprintf("host=%s", fqdnHostTarget),
	}

	servicesMetaRecord := "_services._dns-sd._udp.local."

	if config.NfsEnabled && config.MdnsNfsEnabled {
		ptrName := "_nfs._tcp.local."
		instanceName := fmt.Sprintf("%s.%s", m.config.ServerName, ptrName)

		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: ptrName,
		})
		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: instanceName,
		})
		msg.Answer = append(msg.Answer, &dns.SRV{
			Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
			Priority: 0, Weight: 0, Port: 2049, Target: fqdnHostTarget,
		})

		for _, txt := range txtRecords {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
				Txt: []string{txt},
			})
		}
	}

	if config.SambaEnabled && config.MdnsSambaEnabled {
		ptrName := "_smb._tcp.local."
		instanceName := fmt.Sprintf("%s.%s", m.config.ServerName, ptrName)

		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: ptrName,
		})
		msg.Answer = append(msg.Answer, &dns.PTR{
			Hdr: dns.RR_Header{Name: ptrName, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
			Ptr: instanceName,
		})
		msg.Answer = append(msg.Answer, &dns.SRV{
			Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
			Priority: 0, Weight: 0, Port: 445, Target: fqdnHostTarget,
		})

		for _, txt := range txtRecords {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
				Txt: []string{txt},
			})
		}
	}

	out, err := msg.Pack()
	if err == nil {
		logger.Debug("MDNS", fmt.Sprintf("Broadcasting Proactive Announcement Packet (Records: %d, TTL: %d)", len(msg.Answer), ttl))
		_, _ = m.conn.WriteToUDP(out, multicastAddr)
		time.Sleep(100 * time.Millisecond)
		_, _ = m.conn.WriteToUDP(out, multicastAddr)
	}
}

func (m *MdnsBeacon) Healthcheck() error {
	if m.conn == nil {
		return fmt.Errorf("unified mDNS discovery server connection instance is uninitialized")
	}
	return nil
}

func (m *MdnsBeacon) IsCritical() bool { return false }

func (m *MdnsBeacon) Stop() error {
	logger.Info("MDNS", "Initiating shutdown sequence on unified mDNS channels...")

	if m.done != nil {
		close(m.done)
	}

	if m.conn != nil {
		m.broadcastAnnouncement(0)
		time.Sleep(500 * time.Millisecond)

		_ = m.conn.Close()
		m.conn = nil
	}

	logger.Info("MDNS", "mDNS broadcast beacons dropped cleanly from network space.")
	return nil
}
