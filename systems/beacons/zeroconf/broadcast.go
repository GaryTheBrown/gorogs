package zeroconf

import (
	"time"

	"gorogs/config"

	"github.com/miekg/dns"
)

func (s *Struct) broadcastAnnouncement(ttl uint32) {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Compress = true

	appendDNSAToAnswer(msg, localHostTarget, ttl)
	appendDNSAToAnswer(msg, fqdnHostTarget, ttl)

	if !config.IsDisabled("nfs") && !config.IsDisabled("zeroconf_nfs") {
		appendServiceMetaRecord(msg, nfsAddr, ttl)
		appendServiceData(msg, nfsAddr, 2049, ttl)
	}

	if !config.IsDisabled("samba") && !config.IsDisabled("zeroconf_samba") {
		appendServiceMetaRecord(msg, smbAddr, ttl)
		appendServiceData(msg, smbAddr, 445, ttl)
	}

	out, err := msg.Pack()
	if err == nil {
		_, _ = s.conn.WriteToUDP(out, s.multicastAddr)
		time.Sleep(100 * time.Millisecond)
		_, _ = s.conn.WriteToUDP(out, s.multicastAddr)
	}
}
