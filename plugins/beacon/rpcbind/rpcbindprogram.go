package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/helpers"
	"gorogs/logger"
)

func (s *Struct) startRPCBind() error {
	logger.Debug(Name, "[RPCBIND:STARTING]")

	rpcArgs := []string{"-i"}

	if logger.IsDebugActive(Name) {
		rpcArgs = append(rpcArgs, "-d") // Debug foreground tracking mode remains hot
	}

	containerIPStr := config.SystemIP.String()

	if config.IsDisabled(Name) {
		rpcArgs = append(rpcArgs, "-h", containerIPStr)
	}

	s.rpcCmd = exec.Command(programPath, rpcArgs...)
	s.rpcCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[RPCBIND:CMD SETUP]")

	s.rpcWriter = config.NewSubsystemWriter(Name, nil)
	s.rpcCmd.Stdout = s.rpcWriter
	s.rpcCmd.Stderr = s.rpcWriter
	logger.Debug(Name, "[RPCBIND:LINK STDOUT->LOG]")

	if err := s.rpcCmd.Start(); err != nil {
		if s.rpcWriter != nil {
			_ = s.rpcWriter.Close()
		}
		return fmt.Errorf("failed to execute local rpcbind utility loop: %w", err)
	}
	logger.Debug(Name, "[RPCBIND:START]")

	// Wait for port 111 to stabilize over the internal runtime environment
	if !helpers.WaitForSocket("tcp", "127.0.0.1:111", 5*time.Second) {
		if s.rpcWriter != nil {
			_ = s.rpcWriter.Close()
		}
		return fmt.Errorf("timeout waiting for rpcbind process to open port 111 or process exited early")
	}
	return nil
}

func (s *Struct) stopRPCBind() {
	if s.statdCmd != nil && s.statdCmd.Process != nil {
		logger.Debug(Name, "[RPCBIND:STOPPING]")
		_ = s.statdCmd.Process.Signal(syscall.SIGTERM)
		go func(p *os.Process) {
			time.Sleep(100 * time.Millisecond)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				_ = p.Kill()
			}
		}(s.statdCmd.Process)
		_ = s.statdCmd.Wait()
		logger.Debug(Name, "[RPCBIND:STOPPED]")
	}

	if s.statdWriter != nil {
		_ = s.statdWriter.Close()
	}

}
