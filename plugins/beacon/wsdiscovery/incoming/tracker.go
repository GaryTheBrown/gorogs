package incoming

import (
	"net"
	"strings"
	"sync"
	"time"
)

type PeerSession struct {
	SchemaVersion string
	ExpiresAt     time.Time
}

type SessionTracker struct {
	mu    sync.Mutex
	Store map[string]PeerSession
}

var GlobalSessionTracker = &SessionTracker{
	Store: make(map[string]PeerSession),
}

func extractIP(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	addrStr := addr.String()
	if idx := strings.LastIndex(addrStr, ":"); idx != -1 {
		return addrStr[:idx]
	}
	return addrStr
}

func (s *SessionTracker) RecordSession(sender net.Addr, version string) {
	ipKey := extractIP(sender)
	if ipKey == "" || version == "" {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	for ip, session := range s.Store {
		if now.After(session.ExpiresAt) {
			delete(s.Store, ip)
		}
	}

	s.Store[ipKey] = PeerSession{
		SchemaVersion: version,
		ExpiresAt:     now.Add(2 * time.Minute),
	}
}

func (s *SessionTracker) LookupVersion(sender net.Addr) (string, bool) {
	ipKey := extractIP(sender)
	if ipKey == "" {
		return "", false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	session, found := s.Store[ipKey]
	if !found {
		return "", false
	}

	if time.Now().After(session.ExpiresAt) {
		delete(s.Store, ipKey)
		return "", false
	}

	return session.SchemaVersion, true
}
