package connection

import (
	"context"
	"errors"
	"fmt"
	"gorogs/beacons/wsdd/incoming"
	"gorogs/logger"
	"net"
	"sync"
	"time"
)

func UDPListener(ctx context.Context, discoveryQueue chan<- incoming.WSMessage) (chan struct{}, error) {
	if UDPConn == nil {
		return nil, fmt.Errorf("cannot start receiver: central SharedConn socket is uninitialized")
	}

	var wg sync.WaitGroup
	doneChan := make(chan struct{})

	go ContextMonitor(ctx, UDPConn)
	go PacketReader(ctx, UDPConn, &wg, doneChan, discoveryQueue)

	return doneChan, nil
}

func ContextMonitor(ctx context.Context, conn net.PacketConn) {
	<-ctx.Done()
	logger.Info("wsdd", "Context monitor intercepted master cancellation flag! Closing socket connection immediately...")
	conn.Close()
}

func PacketReader(ctx context.Context, conn net.PacketConn, wg *sync.WaitGroup, doneChan chan struct{}, outputChan chan<- incoming.WSMessage) {
	defer close(doneChan)
	defer wg.Wait()

	logger.Info("wsdd", "Packet reader socket processing loop completely online and consuming buffer blocks")

	for {
		buf := make([]byte, 8192)
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				logger.Info("wsdd", "Packet reader loop broken successfully due to expected system shutdown request")
				break
			}
			logger.Error("wsdd", "Low-level socket interface read operation encountered error", err)
			continue
		}

		packetData := buf[:n]

		if GlobalPacketCache.IsDuplicate(packetData, 3*time.Second) {
			logger.Debug("wsdd", fmt.Sprintf("Suppressed duplicate network packet blast from source: %s", remoteAddr.String()))
			continue
		}

		wg.Add(1)
		logger.Debug("wsdd", fmt.Sprintf("Ingested %d raw packet bytes off network interface wire from sender: %s. Spawning standalone background UDPWorker thread.", n, remoteAddr.String()))
		go UDPWorker(wg, packetData, remoteAddr, outputChan)
	}
}

func UDPWorker(wg *sync.WaitGroup, data []byte, sender net.Addr, outputChan chan<- incoming.WSMessage) {
	defer wg.Done()

	var msg incoming.WSMessage
	senderString := sender.String()

	logger.Debug("wsdd", fmt.Sprintf("[Worker] Commencing strict structural verification and token extraction pass on data from: %s", senderString))
	if FastDecodingMode {
		if err := incoming.QuickDecode(data, &msg); err != nil {
			logger.Error("wsdd", fmt.Sprintf("[Worker] Fast Decoding Failed: %s", senderString), err)
			return
		}
	} else {
		if err := incoming.FullDecode(data, &msg); err != nil {
			logger.Error("wsdd", fmt.Sprintf("[Worker] Dropping invalid/malformed payload block from remote host: %s", senderString), err)
			return
		}
	}

	msg.Sender = sender

	logger.Debug("wsdd", fmt.Sprintf("[Worker] Forwarding successfully verified message packet up to the engine dispatcher queue from sender: %s", senderString))
	outputChan <- msg
}
