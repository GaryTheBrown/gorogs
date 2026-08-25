package nfs

import (
	"fmt"
	"time"

	"gorogs/logger"
	"gorogs/systems/helpers"
)

func (s *Struct) Wait() error {
	if logger.IsDebugActive(s.Name()) {
		select {
		case <-s.readyChan:
			return nil
		case <-time.After(10 * time.Second):
			if s.logWriter != nil {
				_ = s.logWriter.Close()
			}
			return fmt.Errorf("timeout waiting for NFS-Ganesha to declare readiness state milestone log tags")
		}
	} else {
		if !helpers.WaitForSocket("tcp", "127.0.0.1:2049", 10*time.Second) {
			if s.logWriter != nil {
				_ = s.logWriter.Close()
			}
			return fmt.Errorf("timeout waiting for production NFS-Ganesha daemon to bind port 2049")
		}
		return nil
	}
}
