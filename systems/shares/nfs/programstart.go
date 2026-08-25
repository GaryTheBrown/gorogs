package nfs

import (
	"fmt"
	"os/exec"
	"syscall"

	"gorogs/logger"
	"gorogs/systems/helpers"
)

func (s *Struct) StartProgram() error {
	s.readyChan = make(chan struct{})
	ganeshaArgs := []string{"-F", "-f", ganeshaConf}
	if logger.IsDebugActive(s.Name()) {
		ganeshaArgs = append(ganeshaArgs, "-L", "/dev/stdout", "-N", "NIV_FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.DebugAppend(Name, "[CMD SETUP]")

	if logger.IsDebugActive(s.Name()) {
		phrases := []string{"NFS SERVER INITIALIZED", "General fridge was started successfully"}
		s.logWriter = helpers.NewSubsystemWriter(s.Name(), s.readyChan, phrases, nil)
	} else {
		s.logWriter = helpers.NewSubsystemWriter(s.Name(), nil, nil, nil)
	}
	s.ganeshaCmd.Stdout = s.logWriter
	s.ganeshaCmd.Stderr = s.logWriter
	logger.DebugAppend(Name, "[LINK STDOUT->LOG]")

	if err := s.ganeshaCmd.Start(); err != nil {
		close(s.readyChan)
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to initialize background ganesha.nfsd: %w", err)
	}
	logger.DebugAppend(Name, "[CMD START]")

	return nil
}
