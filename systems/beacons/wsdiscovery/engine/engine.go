package engine

import (
	"context"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/connection"
	"gorogs/systems/beacons/wsdiscovery/incoming"
	"gorogs/systems/beacons/wsdiscovery/templates"
)

var (
	Name string
)

type Engine struct {
	DiscoveryQueue chan incoming.WSMessage
	ListenerDone   <-chan struct{}
	ServiceDone    chan struct{}
}

func NewEngineState() *Engine {
	return &Engine{
		DiscoveryQueue: make(chan incoming.WSMessage, 100),
		ServiceDone:    make(chan struct{}),
	}
}

func (s *Engine) Start(ctx context.Context, configDir string) error {
	templates.LoadOrCreatePersistentUUID(configDir, config.Hostname)

	if err := connection.InitUDPSocket(); err != nil {
		logger.Error(Name, "Fatal break: Central WS-Discovery UDP socket failed to bind", err)
		close(s.DiscoveryQueue)
		return err
	}

	if err := connection.InitTCPSocket(s.DiscoveryQueue); err != nil {
		logger.Error(Name, "Fatal break: WS-Transfer TCP socket failed to bind", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}

	doneChan, err := connection.UDPListener(ctx, s.DiscoveryQueue)
	if err != nil {
		logger.Error(Name, "Failed to construct low-level network reader socket", err)
		if connection.UDPConn != nil {
			_ = connection.UDPConn.Close()
		}
		close(s.DiscoveryQueue)
		return err
	}
	s.ListenerDone = doneChan
	go s.ActionDispatcher(ctx)
	s.BroadcastHello()

	return nil
}

func (s *Engine) Stop() {
	s.BroadcastBye()
	<-s.ServiceDone
	close(s.DiscoveryQueue)
}
