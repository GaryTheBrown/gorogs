package nfs

import (
	"fmt"
	"gorogs/logger"
	"net"
	"os"
)

func (s *Struct) SartDummyDBus() error {
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to bind dummy dbus socket listener: %w", err)
	}
	s.dbusListener = listener

	go s.fakeDbusListner()

	return nil
}

func (s *Struct) StopDummyDBus() {
	if s.dbusListener == nil {
		return
	}

	s.dbusListener.Close()
	s.dbusListener = nil

	if rmErr := os.Remove(socketPath); rmErr != nil && !os.IsNotExist(rmErr) {
		logger.WarnF(Name, "Failed to remove dead dbus socket file fragment: %v", rmErr)
	}

	logger.Debug(Name, "[DBUS CLEANUP COMPLETED]")
}

func (s *Struct) fakeDbusListner() {
	for {
		conn, err := s.dbusListener.Accept()
		if err != nil {
			return
		}
		_ = conn.Close()
	}
}
