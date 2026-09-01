package main

import (
	"time"

	"github.com/miekg/dns"
)

func (s *Struct) broadcast(msg *dns.Msg, times uint8, wait time.Duration) error {
	out, err := msg.Pack()
	if err != nil {
		return err
	}

	for range times {
		if _, err = s.conn.WriteToUDP(out, s.multicastAddr); err != nil {
			return err
		}
		if wait > 0 {
			time.Sleep(wait)
		}
	}
	return nil
}

func (s *Struct) broadcastHello() {
	s.broadcast(s.probePacket(), 3, 250*time.Millisecond)

	//conflict stuff here if needed
	s.broadcast(s.packetAnnouncement(), 2, 1*time.Second)
}

func (s *Struct) broadcastPacket() {
	s.broadcast(s.packetAnnouncement(), 1, 0)
}

func (s *Struct) BroadcastBye() {
	msg := s.packetBye()

	if err := s.broadcast(msg, 1, 0); err != nil {
		return
	}

	s.conn.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	defer s.conn.SetReadDeadline(time.Time{})

	buf := make([]byte, 1500)

	for {
		n, _, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			return
		}

		reply := new(dns.Msg)
		if err := reply.Unpack(buf[:n]); err != nil {
			continue
		}

		if reply.Response && len(reply.Extra) > 0 {
			for _, rr := range reply.Extra {
				if rr.Header().Name == hostTarget && rr.Header().Ttl == 0 {
					return
				}
			}
		}
	}
}

func (s *Struct) activeBroadcaster() {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.broadcastPacket()
		}
	}
}
