package main

import (
	"fmt"
	"gorogs/config"

	"github.com/miekg/dns"
)

func (s *Struct) probePacket() *dns.Msg {
	msg := new(dns.Msg)
	msg.Response = false
	msg.Opcode = dns.OpcodeQuery
	msg.Id = 0

	appendQuestion(msg, dnsQuestion(hostTarget))
	appendNS(msg, dnsAProbe())

	for _, addr := range Addrs {
		if !addr.Enabled {
			continue
		}
		appendQuestion(msg, dnsQuestion(fmt.Sprintf("%s.%s", config.Hostname, addr.Address)))
		appendNS(msg, dnsSRVProbe(addr))

	}
	return msg
}

func (s *Struct) packetAnnouncement() *dns.Msg {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Id = 0

	appendExtra(msg, dnsA(), dnsNSECHost())

	for _, addr := range Addrs {
		if !addr.Enabled {
			continue
		}
		appendAnswer(msg,
			dnsPTRMetaRecord(addr.Address),
			dnsPTRService(addr.Address),
		)
		appendExtra(msg, dnsSRV(addr))
		if singleTextRecord {
			appendExtra(msg, dnsTXTSingle(addr.Address))
		} else {
			appendExtra(msg, dnsTXTMultiple(addr.Address)...)
		}
		appendExtra(msg, dnsNSECServiceInstance(addr.Address))

	}

	return msg
}

func (s *Struct) packetSingleSystem(addr AddrType) *dns.Msg {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Id = 0

	appendExtra(msg,
		dnsA(),
		dnsNSECHost(),
	)

	appendAnswer(msg,
		dnsPTRMetaRecord(addr.Address),
		dnsPTRService(addr.Address),
	)
	appendExtra(msg, dnsSRV(addr))
	if singleTextRecord {
		appendExtra(msg, dnsTXTSingle(addr.Address))
	} else {
		appendExtra(msg, dnsTXTMultiple(addr.Address)...)
	}
	appendExtra(msg, dnsNSECServiceInstance(addr.Address))

	return msg
}

func (s *Struct) packetBye() *dns.Msg {
	msg := new(dns.Msg)
	msg.Response = true
	msg.Authoritative = true
	msg.Id = 0

	appendExtra(msg,
		dnsA(0),
		dnsNSECHost(0),
	)

	for _, addr := range Addrs {
		if !addr.Enabled {
			continue
		}
		appendAnswer(msg,
			dnsPTRMetaRecord(addr.Address, 0),
			dnsPTRService(addr.Address, 0),
		)
		appendExtra(msg, dnsSRV(addr, 0))
		if singleTextRecord {
			appendExtra(msg, dnsTXTSingle(addr.Address, 0))
		} else {
			appendExtra(msg, dnsTXTMultiple(addr.Address, 0)...)
		}
		appendExtra(msg, dnsNSECServiceInstance(addr.Address, 0))
	}

	return msg
}
