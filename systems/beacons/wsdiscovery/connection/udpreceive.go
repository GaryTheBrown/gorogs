package connection

import (
	"context"
	"errors"
	"fmt"
	"gorogs/logger"
	"gorogs/systems/beacons/wsdiscovery/incoming"
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
	conn.Close()
}

func PacketReader(ctx context.Context, conn net.PacketConn, wg *sync.WaitGroup, doneChan chan struct{}, outputChan chan<- incoming.WSMessage) {
	defer close(doneChan)
	defer wg.Wait()

	for {
		buf := make([]byte, 8192)
		n, remoteAddr, err := conn.ReadFrom(buf)
		if err != nil {
			if errors.Is(err, net.ErrClosed) || ctx.Err() != nil {
				break
			}
			logger.Error(Name, "Low-level socket interface read operation encountered error", err)
			continue
		}

		packetData := buf[:n]

		if GlobalPacketCache.IsDuplicate(packetData, 3*time.Second) {
			continue
		}

		wg.Add(1)
		go UDPWorker(wg, packetData, remoteAddr, outputChan)
	}
}

func UDPWorker(wg *sync.WaitGroup, data []byte, sender net.Addr, outputChan chan<- incoming.WSMessage) {
	defer wg.Done()
	var msg incoming.WSMessage
	senderString := sender.String()
	if FastDecodingMode {
		if err := incoming.QuickDecode(data, &msg); err != nil {
			logger.ErrorF(Name, "[Worker] Fast Decoding Failed: %s", err, senderString)
			return
		}
	} else {
		if err := incoming.FullDecode(data, &msg); err != nil {
			logger.ErrorF(Name, "[Worker] Dropping invalid/malformed payload block from remote host: %s", err, senderString)
			return
		}
	}

	msg.Sender = sender
	outputChan <- msg
}
