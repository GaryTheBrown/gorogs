package zeroconf

import (
	"fmt"

	"gorogs/config"

	"github.com/miekg/dns"
)

func appendDNSAToAnswer(resp *dns.Msg, name string, ttl uint32) {
	resp.Answer = append(resp.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   config.SystemIP,
	})
}

func appendDNSAToExtra(resp *dns.Msg, name string, ttl uint32) {
	resp.Extra = append(resp.Extra, &dns.A{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: ttl},
		A:   config.SystemIP,
	})
}

func appendServiceMetaRecord(resp *dns.Msg, name string, ttl uint32) {
	resp.Answer = append(resp.Answer, &dns.PTR{
		Hdr: dns.RR_Header{Name: servicesMetaRecord, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
		Ptr: name,
	})
}

func appendServiceData(resp *dns.Msg, name string, port uint16, ttl uint32) {
	instanceName := fmt.Sprintf("%s.%s", config.Hostname, name)
	resp.Answer = append(resp.Answer, &dns.PTR{
		Hdr: dns.RR_Header{Name: name, Rrtype: dns.TypePTR, Class: dns.ClassINET, Ttl: ttl},
		Ptr: instanceName,
	})
	resp.Extra = append(resp.Extra, &dns.SRV{
		Hdr:      dns.RR_Header{Name: instanceName, Rrtype: dns.TypeSRV, Class: dns.ClassINET, Ttl: ttl},
		Priority: 0, Weight: 0, Port: port, Target: fqdnHostTarget,
	})
	for _, txt := range txtRecords {
		resp.Extra = append(resp.Extra, &dns.TXT{
			Hdr: dns.RR_Header{Name: instanceName, Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: ttl},
			Txt: []string{txt},
		})
	}
}
