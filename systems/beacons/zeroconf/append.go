package zeroconf

import (
	"github.com/miekg/dns"
)

func appendQuestion(msg *dns.Msg, questions ...*dns.Question) {
	for _, q := range questions {
		if q != nil {
			msg.Question = append(msg.Question, *q)
		}
	}
}

func appendAnswer(msg *dns.Msg, record ...dns.RR) {
	msg.Answer = append(msg.Answer, record...)
}

func appendNS(msg *dns.Msg, record ...dns.RR) {
	msg.Ns = append(msg.Ns, record...)
}

func appendExtra(msg *dns.Msg, record ...dns.RR) {
	msg.Extra = append(msg.Extra, record...)
}
