package engine

import (
	"context"
	"gorogs/systems/beacons/wsdiscovery/versions"
)

func (s *Engine) ActionDispatcher(ctx context.Context) {
	defer close(s.ServiceDone)
	for msg := range s.DiscoveryQueue {
		switch msg.Header.ActionType {
		case versions.Probe:
			s.ExecuteProbeAction(msg)
		case versions.Resolve:
			s.ExecuteResolveAction(msg)
		case versions.Get:
			s.ExecuteGetAction(msg)
		default:
		}
	}

	<-s.ListenerDone
}
