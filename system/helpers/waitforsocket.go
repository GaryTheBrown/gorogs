package helpers

import (
	"errors"
	"net"
	"os"
	"strings"
	"syscall"
	"time"
)

func WaitForSocket(network, address string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if strings.HasPrefix(network, "udp") {
			packetConn, err := net.ListenPacket(network, address)
			if err != nil {
				if strings.Contains(err.Error(), "address already in use") {
					return true
				}
			} else {
				packetConn.Close()
			}
		} else {
			conn, err := net.DialTimeout(network, address, 100*time.Millisecond)
			if err == nil {
				conn.Close()
				return true
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return false
}

type ProcessStateProvider interface {
	Signal(sig os.Signal) error
}

// WaitForDaemonReady adapts to massive file allocations using process awareness and backoff scaling
func WaitForDaemonReady(process ProcessStateProvider, address string, maxWait time.Duration) error {
	// Configure adaptive backoff limits
	minTick := 100 * time.Millisecond
	maxTick := 3000 * time.Millisecond
	currentTick := minTick

	startTime := time.Now()
	probeAttempts := 0

	for time.Since(startTime) < maxWait {
		probeAttempts++

		// 1. LIVENESS CHECK: Verify if the daemon process is still running natively
		// Sending signal 0 checks for process existence without actually terminating it
		if err := process.Signal(syscall.Signal(0)); err != nil {
			return errors.New("underlying daemon process terminated abnormally before binding ports")
		}

		// 2. NETWORK PROBE: Attempt a light dial connection check against the target socket
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
			return nil // Success: The server is fully awake and listening!
		}

		// 3. ADAPTIVE BACKOFF: Gradually slow down checking frequency to conserve host CPU
		if probeAttempts > 10 {
			// Multiply sleep duration by 1.5x up to our hard capped limit
			currentTick = min(time.Duration(float64(currentTick)*1.5), maxTick)
		}

		time.Sleep(currentTick)
	}

	return errors.New("timeout reached waiting for subsystem socket synchronization")
}
