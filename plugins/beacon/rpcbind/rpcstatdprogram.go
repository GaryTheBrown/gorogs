package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
)

func (s *Struct) startRPCStatd() error {
	statdArgs := []string{
		"-F",
		"-n", config.Hostname,
		"-p", "32765",
		"-o", "32766",
	}

	s.statdCmd = exec.Command(statdPath, statdArgs...)
	s.statdCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[RPC.STATD:CMD SETUP]")

	s.statdWriter = config.NewSubsystemWriter(Name, nil)
	s.statdCmd.Stdout = s.statdWriter
	s.statdCmd.Stderr = s.statdWriter
	logger.Debug(Name, "[RPC.STATD:LINK STDOUT->LOG]")

	if err := s.statdCmd.Start(); err != nil {
		logger.Error(Name, "Failed to launch network status monitor process tree", err)
		if s.statdWriter != nil {
			_ = s.statdWriter.Close()
		}
		return fmt.Errorf("failed to start rpc.statd: %w", err)
	}
	logger.Debug(Name, "[RPC.STATD:START]")
	return nil
}

func (s *Struct) stopRPCStatd() {
	if s.rpcCmd != nil && s.rpcCmd.Process != nil {
		logger.Debug(Name, "[RPC.STATD:STOPPING]")
		_ = s.rpcCmd.Process.Signal(syscall.SIGTERM)
		go func(p *os.Process) {
			time.Sleep(100 * time.Millisecond)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				_ = p.Kill()
			}
		}(s.rpcCmd.Process)
		_ = s.rpcCmd.Wait()
		logger.Debug(Name, "[RPC.STATD:STOPPED]")
	}

	if s.rpcWriter != nil {
		_ = s.rpcWriter.Close()
	}
}
