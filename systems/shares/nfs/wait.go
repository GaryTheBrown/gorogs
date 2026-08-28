package nfs

import (
	"fmt"
	"net"
	"time"
)

func (s *Struct) Wait() error {
	timeout := time.Now().Add(10 * time.Second)
	for time.Now().Before(timeout) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:2049", 100*time.Millisecond)
		if err == nil {
			conn.Close()
			close(s.readyChan)
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	close(s.readyChan)
	return fmt.Errorf("timeout waiting for background NFS daemon to listen on port 2049")
}
