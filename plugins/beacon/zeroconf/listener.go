package main

import (
	"fmt"
	"gorogs/config"
	"strings"

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

			if msg.Response || remoteAddr.IP.Equal(config.SystemIP) {
				continue
			}

			resp := new(dns.Msg)
			resp.SetReply(msg)
			resp.Compress = true
			resp.Authoritative = true
			resp.Id = 0
			resp.Question = nil

			hasAnswers := false
			isUnicastTarget := false

			for _, question := range msg.Question {
				if (question.Qclass & 0x8000) != 0 {
					isUnicastTarget = true
				}

				qClass := question.Qclass & 0x7FFF
				if qClass != dns.ClassINET {
					continue
				}

				switch question.Name {

				case servicesMetaRecord:
					if question.Qtype == dns.TypePTR || question.Qtype == dns.TypeANY {
						for _, addr := range Addrs {
							if !addr.Enabled {
								continue
							}
							appendAnswer(resp, dnsPTRMetaRecord(addr.Address))
							hasAnswers = true
						}
					}

				case hostTarget:
					if question.Qtype == dns.TypeA || question.Qtype == dns.TypeANY {
						appendAnswer(resp, dnsA())
						appendExtra(resp, dnsNSECHost())
						hasAnswers = true
					}

				default:
					for _, addr := range Addrs {
						if !addr.Enabled {
							continue
						}

						if question.Name == addr.Address && (question.Qtype == dns.TypePTR || question.Qtype == dns.TypeANY) {
							appendAnswer(resp, dnsPTRService(addr.Address))
							appendExtra(resp, dnsSRV(addr))
							if singleTextRecord {
								appendExtra(resp, dnsTXTSingle(addr.Address))
							} else {
								appendExtra(resp, dnsTXTMultiple(addr.Address)...)
							}
							appendExtra(resp, dnsNSECServiceInstance(addr.Address))
							hasAnswers = true
							break
						}

						instanceFQDN := fmt.Sprintf("%s.%s", config.Hostname, addr.Address)
						if strings.EqualFold(question.Name, instanceFQDN) {
							if question.Qtype == dns.TypeSRV || question.Qtype == dns.TypeANY {
								appendAnswer(resp, dnsSRV(addr))
								hasAnswers = true
							}
							if question.Qtype == dns.TypeTXT || question.Qtype == dns.TypeANY {
								if singleTextRecord {
									appendExtra(resp, dnsTXTSingle(addr.Address))
								} else {
									appendExtra(resp, dnsTXTMultiple(addr.Address)...)
								}
								hasAnswers = true
							}
						}
					}
				}
			}

			if hasAnswers {
				out, err := resp.Pack()
				if err != nil {
					continue
				}

				if isUnicastTarget {
					_, _ = s.conn.WriteToUDP(out, remoteAddr)
				} else {
					_, _ = s.conn.WriteToUDP(out, s.multicastAddr)
				}
			}
		}
	}
}
