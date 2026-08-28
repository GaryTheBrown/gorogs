package samba

import (
	"fmt"
	"os/exec"
	"syscall"

	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/shares/samba/vars"
)

func (s *Struct) ProgramStart() error {

	args := []string{"--foreground", "--no-process-group", "-s", vars.MasterConfigFile, "--debug-stdout"}

	if logger.IsDebugActive(Name) {
		args = append(args, "-d", "3")
	} else {
		args = append(args, "-d", "0")
	}
	vars.Cmd = exec.Command(vars.ProgramPath, args...)
	vars.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[COMMAND ARGS]")

	if logger.IsDebugActive(Name) {
		s.logWriter = helpers.NewSubsystemWriter(Name, nil)
		vars.Cmd.Stdout = s.logWriter
		vars.Cmd.Stderr = s.logWriter
		logger.Debug(Name, "[ATTACHING LOGS]")
	}

	if err := vars.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background smbd process: %w", err)
	}
	logger.Debug(Name, "[CMD START]")

	return nil
}
