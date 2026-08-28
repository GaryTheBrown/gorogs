package rpcbind

import (
	"os"
	"os/exec"
	"syscall"
	"time"

	"gorogs/logger"
	"gorogs/systems/helpers"
)

func (s *Struct) startRPCStatd() error {

	s.statdCmd = exec.Command(statdPath, "-F")
	s.statdCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.DebugAppend(Name, "[RPC.STATD:CMD SETUP]")
	if logger.IsDebugActive(Name) {
		s.statdWriter = helpers.NewSubsystemWriter(Name, nil)
		s.statdCmd.Stdout = s.statdWriter
		s.statdCmd.Stderr = s.statdWriter
		logger.DebugAppend(Name, "[RPC.STATD:LINK STDOUT->LOG]")
	}

	if err := s.statdCmd.Start(); err != nil {
		logger.Error(Name, "Failed to launch network status monitor process tree", err)
		if s.statdWriter != nil {
			_ = s.statdWriter.Close()
		}
	}
	logger.DebugAppend(Name, "[RPC.STATD:START]")
	return nil
}

func (s *Struct) stopRPCStatd() {
	if s.rpcCmd != nil && s.rpcCmd.Process != nil {
		logger.DebugAppend(Name, "[RPC.STATD:STOPPING]")
		_ = s.rpcCmd.Process.Signal(syscall.SIGTERM)
		go func(p *os.Process) {
			time.Sleep(100 * time.Millisecond)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				_ = p.Kill()
			}
		}(s.rpcCmd.Process)
		_ = s.rpcCmd.Wait()
		logger.DebugAppend(Name, "[RPC.STATD:STOPPED]")
	}

	if s.rpcWriter != nil {
		_ = s.rpcWriter.Close()
	}
}
