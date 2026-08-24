package samba

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/shares/samba/modes"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "Samba"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

type Struct struct {
	sState systeminterface.SysStateEnum

	logWriter   *helpers.SubsystemWriter
	readyChan   chan struct{}
	cancelWatch context.CancelFunc
	sys         modes.System
	shares      structs.ShareMap
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

// DEFAULT CONFIG VARS TO EVENTUALLY BE LOADED IN FORM A map[string]any
var (
	systemMode = structs.ModeRegistry
)

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	if vars.LibBaseDirOverlay {
		if err := s.setupOverlay(); err != nil {
			logger.Fatal(Name, "failed to execute master config write utility", err)
		}
	}

	modeStr := structs.ModeToString(systemMode)
	logger.DebugAppendF(Name, "[MODE:%s]", modeStr)
	cm := modes.SharedConfigFile(systemMode)
	sm := structs.NewShareMap()
	logger.DebugAppendF(Name, "[SHARES: Count(%d)]", sm.Count())
	switch systemMode {
	case structs.ModeFile:
		s.sys = &modes.ModeFile{
			ConfigMap: cm,
			SharesMap: sm,
		}
	case structs.ModeMixed:
		s.sys = &modes.ModeMixed{
			ConfigMap: cm,
			SharesMap: sm,
		}
	case structs.ModeRegistry:
		s.sys = &modes.ModeRegistry{
			ConfigMap: cm,
			SharesMap: sm,
		}
	default:
		logger.FatalF(Name, "failed to get Sambas system mode. Got Int %d", nil, int(systemMode))
	}

	logger.DebugAppendF(Name, "[MODE %s SETUP]", modeStr)
	if err := s.sys.Setup(); err != nil {
		logger.FatalF(Name, "failed to Setup the Mode [%s]", err, modeStr)
	}

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")

	args := []string{"--foreground", "--no-process-group", "-s", vars.MasterConfigFile, "--debug-stdout"}

	if logger.IsDebugActive(Name) {
		args = append(args, "-d", "3")
	} else {
		args = append(args, "-d", "0")
	}
	vars.Cmd = exec.Command(vars.ProgramPath, args...)
	vars.Cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.DebugAppend(Name, "[COMMAND ARGS]")

	if logger.IsDebugActive(Name) {
		s.logWriter = helpers.NewSubsystemWriter(Name, nil, nil, nil)
		vars.Cmd.Stdout = s.logWriter
		vars.Cmd.Stderr = s.logWriter
		logger.DebugAppend(Name, "[ATTACHING LOGS]")
	}

	if err := vars.Cmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background smbd process: %w", err)
	}
	logger.DebugAppend(Name, "[CMD START]")

	portBound := false
	maxWait := 20 * time.Second
	currentTick := 100 * time.Millisecond
	startTime := time.Now()
	probeAttempts := 0
	logger.DebugAppend(Name, "[WAIT")
	for time.Since(startTime) < maxWait {
		probeAttempts++
		logger.DebugAppend(Name, ".")
		if err := vars.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
			if s.logWriter != nil {
				s.logWriter.Close()
			}
			return fmt.Errorf("samba smbd daemon process terminated unexpectedly during boot")
		}
		if helpers.WaitForSocket("tcp", "127.0.0.1:445", 50*time.Millisecond) {
			portBound = true
			break
		}
		if probeAttempts > 10 {
			currentTick = min(time.Duration(float64(currentTick)*1.5), 2*time.Second)
		}
		time.Sleep(currentTick)
	}
	logger.DebugAppend(Name, "DONE]")

	if !portBound {
		if s.logWriter != nil {
			s.logWriter.Close()
		}
		return fmt.Errorf("timeout reached waiting for Samba daemon to bind network port 445")
	}
	logger.DebugAppend(Name, "[READY]")

	if !config.IsDisabled("livechanges") {
		watchCtx, cancel := context.WithCancel(context.Background())
		s.cancelWatch = cancel
		go s.startFSEventDirectoryWatcher(watchCtx)
		logger.DebugAppend(Name, "[TRACKING]")
	}

	s.sState = systeminterface.STARTED
	logger.DebugEnd(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.DebugContinue(Name, "Stopping NetBIOS daemon threads...")
	if s.cancelWatch != nil {
		s.cancelWatch()
		logger.DebugAppend(Name, "[CANCEL WATCH]")
	}

	if vars.Cmd != nil && vars.Cmd.Process != nil {

		if err := vars.Cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = vars.Cmd.Process.Kill()
		}
		logger.DebugAppend(Name, "[KILL SENT]")
		_ = vars.Cmd.Wait()
		logger.DebugAppend(Name, "[STOPPED]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.DebugAppend(Name, "[DETACH LOGS]")
	}

	if vars.LibBaseDirOverlay {
		if err := syscall.Unmount(vars.SambaBaseLibDir, 0); err != nil {
			logger.ErrorF(Name, "Failed to unmount memory partition layer cleanly from layout ERROR: %v", err, err.Error())
		}
		logger.DebugAppend(Name, "[REMOVE OVERLAY]")
	}

	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if vars.Cmd == nil || vars.Cmd.Process == nil {
		return fmt.Errorf("samba background system execution tracking instance is not initialized")
	}
	return vars.Cmd.Process.Signal(syscall.Signal(0))
}
