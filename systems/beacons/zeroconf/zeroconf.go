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

type ZeroconfStruct struct {
	sState systeminterface.SysStateEnum
	conn   *net.UDPConn
	done   chan struct{}
}

func (_ ZeroconfStruct) Name() string                                { return "mdns" }
func (_ ZeroconfStruct) Type() systeminterface.SystemTypeEnum        { return systeminterface.Beacon }
func (_ ZeroconfStruct) IsCritical() bool                            { return false }
func (_ ZeroconfStruct) AutoStart() bool                             { return true }
func (z *ZeroconfStruct) State(in systeminterface.SysStateEnum) bool { return z.sState == in }

func (z *ZeroconfStruct) Healthcheck() error {
	if z.conn == nil {
		return fmt.Errorf("unified mDNS discovery server connection instance is uninitialized")
	}
	return nil
}

func (z *ZeroconfStruct) Setup() {
	// logger.Info(m.Name(), "Evaluating network discovery broadcast requirements...")
	z.sState = systeminterface.SETUP
	// logger.Info(m.Name(), "Pre-flight checks passed. Service registration profile is valid.")
}

func (z *ZeroconfStruct) Start() error {
	logger.Info(z.Name(), "Initializing proactive proxy-routing mDNS discovery engine...")

	multicastAddr, err := net.ResolveUDPAddr("udp4", "224.0.0.251:5353")
	if err != nil {
		return fmt.Errorf("failed to resolve multicast address block: %w", err)
	}

	conn, err := net.ListenMulticastUDP("udp4", nil, multicastAddr)
	if err != nil {
		return fmt.Errorf("failed to join kernel mDNS multicast loop: %w", err)
	}
	z.conn = conn
	z.done = make(chan struct{})

	z.broadcastAnnouncement(120)

	go z.listenForQueries()

	logger.Info(z.Name(), "Universal mDNS service discovery proxies active and broadcasting Hello packets.")
	z.sState = systeminterface.STARTED
	return nil
}

func (z *ZeroconfStruct) listenForQueries() {
	localHostTarget := fmt.Sprintf("%s.local.", config.Hostname)
	fqdnHostTarget := fmt.Sprintf("%s.%s.", config.Hostname, config.DomainName)

	servicesMetaRecord := "_services._dns-sd._udp.local."
	txtRecords := []string{"path=/", fmt.Sprintf("host=%s", fqdnHostTarget)}

	buf := make([]byte, 1500)
	for {
		select {
		case <-z.done:
			return
		default:
			n, remoteAddr, err := z.conn.ReadFromUDP(buf)
			if err != nil {
				return
			}

			msg := new(dns.Msg)
			if err := msg.Unpack(buf[:n]); err != nil {
				continue
			}

			for _, q := range msg.Question {
				logger.DebugF(z.Name(), "Incoming Query from %s: Name=%s Type=%s", remoteAddr.String(), q.Name, dns.TypeToString[q.Qtype])

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

					if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") && (q.Name == servicesMetaRecord || q.Name == "_nfs._tcp.local.") {
						ptrName := "_nfs._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

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

					if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") && (q.Name == servicesMetaRecord || q.Name == "_smb._tcp.local.") {
						ptrName := "_smb._tcp.local."
						instanceName := fmt.Sprintf("%s.%s", config.Hostname, ptrName)

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
						logger.DebugF(z.Name(), "Sending Targeted Response to %s for %s", remoteAddr.String(), q.Name)
						_, _ = z.conn.WriteToUDP(out, remoteAddr)
					}
				}
			}
		}
	}
}

func (z *ZeroconfStruct) broadcastAnnouncement(ttl uint32) {
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
		logger.DebugF(z.Name(), "Broadcasting Proactive Announcement Packet (Records: %d, TTL: %d)", len(msg.Answer), ttl)
		_, _ = z.conn.WriteToUDP(out, multicastAddr)
		time.Sleep(100 * time.Millisecond)
		_, _ = z.conn.WriteToUDP(out, multicastAddr)
	}
}

func (z *ZeroconfStruct) Stop() {
	logger.Info(z.Name(), "Initiating shutdown sequence on unified mDNS channels...")

	if z.done != nil {
		close(z.done)
	}

	if z.conn != nil {
		z.broadcastAnnouncement(0)
		time.Sleep(500 * time.Millisecond)

		_ = z.conn.Close()
		z.conn = nil
	}

	logger.Info(z.Name(), "mDNS broadcast beacons dropped cleanly from network space.")
	z.sState = systeminterface.STOPPED
}
