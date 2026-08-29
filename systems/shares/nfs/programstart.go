package nfs

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"

	"gorogs/logger"
	"gorogs/systems/helpers"
)

func (s *Struct) StartProgram() error {
	s.readyChan = make(chan struct{})
	ganeshaArgs := []string{"-F", "-f", ganeshaConf, "-L", "/dev/stdout"}
	if logger.IsDebugActive(Name) {
		ganeshaArgs = append(ganeshaArgs, "-x", "-N", "FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	if s.zeroFreeSpace {
		s.ganeshaCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libfake_statfs.so")
		logger.Debug(Name, "[ZERO FREE SPACE ENABLED]")
	}

	s.logWriter = helpers.NewSubsystemWriter(Name, nil)
	s.ganeshaCmd.Stdout = s.logWriter
	s.ganeshaCmd.Stderr = s.logWriter
	logger.Debug(Name, "[LINK STDOUT->LOG]")

	if err := s.ganeshaCmd.Start(); err != nil {
		close(s.readyChan)
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to initialize background ganesha.nfsd: %w", err)
	}
	logger.Debug(Name, "[CMD START]")

	return nil
}
