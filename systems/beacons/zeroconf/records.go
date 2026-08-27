package zeroconf

import (
	"fmt"
	"gorogs/config"

	"github.com/miekg/dns"
)

func headerDontFlush(rrtype uint16, name string, ttl uint32) dns.RR_Header {
	return dns.RR_Header{
		Name:   name,
		Rrtype: rrtype,
		Class:  dns.ClassINET,
		Ttl:    ttl,
	}
}

func headerFlush(rrtype uint16, name string, ttl uint32) dns.RR_Header {
	return dns.RR_Header{
		Name:   name,
		Rrtype: rrtype,
		Class:  dns.ClassINET | 0x8000,
		Ttl:    ttl,
	}
}

func dnsQuestion(name string) *dns.Question {
	return &dns.Question{
		Name:   name,
		Qtype:  dns.TypeANY,
		Qclass: dns.ClassINET,
	}
}

func dnsPTRService(serviceAddr string, ttl ...uint32) *dns.PTR {
	ttlVal := ttlShared
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	return &dns.PTR{
		Hdr: headerDontFlush(dns.TypePTR, serviceAddr, ttlVal),
		Ptr: fmt.Sprintf("%s.%s", config.Hostname, serviceAddr),
	}
}

func dnsPTRMetaRecord(serviceType string, ttl ...uint32) *dns.PTR {
	ttlVal := ttlShared
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	return &dns.PTR{
		Hdr: headerDontFlush(dns.TypePTR, servicesMetaRecord, ttlVal),
		Ptr: serviceType,
	}
}
func dnsA(ttl ...uint32) *dns.A {
	ttlVal := ttlUnique
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	return &dns.A{
		Hdr: headerFlush(dns.TypeA, hostTarget, ttlVal),
		A:   config.SystemIP,
	}
}

func dnsSRV(service AddrType, ttl ...uint32) *dns.SRV {
	ttlVal := ttlUnique
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	return &dns.SRV{
		Hdr:      headerFlush(dns.TypeSRV, fmt.Sprintf("%s.%s", config.Hostname, service.Address), ttlVal),
		Priority: 0,
		Weight:   0,
		Port:     service.Port,
		Target:   hostTarget,
	}
}

func dnsTXTSingle(serviceAddr string, ttl ...uint32) *dns.TXT {
	ttlVal := ttlShared
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	txtRecordsList := []string{
		"path=/",
		fmt.Sprintf("host=%s", hostParam),
		fmt.Sprintf("model=%s", serverIcon),
	}
	return &dns.TXT{
		Hdr: headerFlush(dns.TypeTXT, fmt.Sprintf("%s.%s", config.Hostname, serviceAddr), ttlVal),
		Txt: txtRecordsList,
	}
}

func dnsTXTMultiple(serviceAddr string, ttl ...uint32) []dns.RR {
	ttlVal := ttlShared
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}

	instanceName := fmt.Sprintf("%s.%s", config.Hostname, serviceAddr)

	txtRecordsList := []string{
		"path=/",
		fmt.Sprintf("host=%s", hostParam),
		fmt.Sprintf("model=%s", serverIcon),
	}

	var records []dns.RR

	for _, txtStr := range txtRecordsList {
		records = append(records, &dns.TXT{
			Hdr: headerFlush(dns.TypeTXT, instanceName, ttlVal),
			Txt: []string{txtStr},
		})
	}

	return records
}

func dnsNSEC(name string, typeBitMask []uint16, ttl ...uint32) *dns.NSEC {
	ttlVal := ttlUnique
	if len(ttl) > 0 {
		ttlVal = ttl[0]
	}
	return &dns.NSEC{
		Hdr:        headerFlush(dns.TypeNSEC, name, ttlVal),
		NextDomain: name,
		TypeBitMap: typeBitMask,
	}
}

func dnsNSECHost(ttl ...uint32) *dns.NSEC {
	return dnsNSEC(hostTarget, []uint16{dns.TypeA, dns.TypeNSEC}, ttl...)
}

func dnsNSECServiceInstance(serviceAddr string, ttl ...uint32) *dns.NSEC {
	return dnsNSEC(fmt.Sprintf("%s.%s", config.Hostname, serviceAddr), []uint16{dns.TypeTXT, dns.TypeSRV, dns.TypeNSEC}, ttl...)
}

func dnsAProbe() *dns.A {
	return &dns.A{
		Hdr: headerDontFlush(dns.TypeA, hostTarget, ttlProbe),
		A:   config.SystemIP,
	}
}

func dnsSRVProbe(service AddrType) *dns.SRV {
	return &dns.SRV{
		Hdr:      headerDontFlush(dns.TypeSRV, fmt.Sprintf("%s.%s", config.Hostname, service.Address), ttlProbe),
		Priority: 0,
		Weight:   0,
		Port:     service.Port,
		Target:   hostTarget,
	}
}
