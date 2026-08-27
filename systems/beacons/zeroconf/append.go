package zeroconf

import (
	"github.com/miekg/dns"
)

func appendQuestion(msg *dns.Msg, question *dns.Question) {
	msg.Question = append(msg.Question, *question)
}

func appendAnswer(msg *dns.Msg, record dns.RR) {
	msg.Answer = append(msg.Answer, record)
}

func appendNS(msg *dns.Msg, record dns.RR) {
	msg.Ns = append(msg.Ns, record)
}

func appendExtra(msg *dns.Msg, record dns.RR) {
	msg.Extra = append(msg.Extra, record)
}
