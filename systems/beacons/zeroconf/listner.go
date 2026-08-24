package zeroconf

import (
	"gorogs/config"

	"github.com/miekg/dns"
)

func (s *Struct) listenForQueries() {

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
				if q.Name == localHostTarget || q.Name == fqdnHostTarget || q.Name == servicesMetaRecord || q.Name == nfsAddr || q.Name == smbAddr {
					resp := new(dns.Msg)
					resp.SetReply(msg)
					resp.Compress = true
					resp.Authoritative = true

					if q.Name == localHostTarget || q.Name == fqdnHostTarget {
						appendDNSAToAnswer(msg, localHostTarget, 120)
						appendDNSAToAnswer(msg, fqdnHostTarget, 120)
					} else {
						appendDNSAToExtra(msg, localHostTarget, 120)
						appendDNSAToExtra(msg, fqdnHostTarget, 120)
					}

					if q.Name == servicesMetaRecord {
						if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") {
							appendServiceMetaRecord(resp, nfsAddr, 120)
						}
						if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") {
							appendServiceMetaRecord(resp, smbAddr, 120)
						}
					}

					if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") && q.Name == nfsAddr {
						appendServiceData(resp, nfsAddr, 2049, 120)
					}

					if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") && q.Name == smbAddr {
						appendServiceData(resp, smbAddr, 445, 120)
					}

					out, err := resp.Pack()
					if err == nil {
						_, _ = s.conn.WriteToUDP(out, remoteAddr)
					}
				}
			}
		}
	}
}
