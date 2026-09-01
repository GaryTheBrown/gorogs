package engine

import (
	"context"
	"gorogs/plugins/beacon/wsdiscovery/versions"
)

func (e *Engine) ActionDispatcher(ctx context.Context) {
	defer close(e.ServiceDone)
	for msg := range e.DiscoveryQueue {
		switch msg.Header.ActionType {
		case versions.Probe:
			e.ExecuteProbeAction(msg)
		case versions.Resolve:
			e.ExecuteResolveAction(msg)
		case versions.Get:
			e.ExecuteGetAction(msg)
		default:
		}
	}

	<-e.ListenerDone
}
