package samba

import (
	"context"
	"fmt"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/shares/samba/modes"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
	"gorogs/systems/systeminterface"

	"github.com/fsnotify/fsnotify"
)

const (
	Name       = "Samba"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

type Struct struct {
	sState systeminterface.SysStateEnum

	logWriter      *helpers.SubsystemWriter
	readyChan      chan struct{}
	cancelWatch    context.CancelFunc
	sys            modes.System
	shares         structs.ShareMap
	commentWatcher *fsnotify.Watcher
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

var (
	systemMode = structs.ModeRegistry
)

func (s *Struct) Config() {
	cm := config.GetServiceConfig(Name)
	mode := cm.Get("mode", "Registry")
	systemMode = structs.StringToMode(mode)
	vars.BaseDirOverlay = cm.Get("basediroverlay", true)
	vars.BatchInjection = cm.Get("batchinjection", true)
	vars.VetoFiles = cm.Get("vetofiles", "/*.~tmp/")
	vars.DefaultShareComment = cm.Get("defaultcomment", "")
	vars.ServerComment = cm.Get("servercomment", "GO Read Only Guest Share")
}

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	if vars.BaseDirOverlay {
		if err := s.setupOverlay(); err != nil {
			logger.Fatal(Name, "failed to execute master config write utility", err)
		}
	}

	s.setupSystem()

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")

	if err := s.ProgramStart(); err != nil {
		return err
	}

	if !s.WaitForStart(20 * time.Second) {
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
		go s.startFSEventCommentWatcher(watchCtx)
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
		logger.DebugAppend(Name, "[SIGTERM SENT]")

		// Ensure the parent daemon tracker releases its process hooks completely
		_ = vars.Cmd.Wait()
		logger.DebugAppend(Name, "[PROCESS TERMINATED]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.DebugAppend(Name, "[DETACH LOGS]")
	}

	if vars.BaseDirOverlay {
		if err := syscall.Unmount(vars.SambaBaseLibDir, syscall.MNT_DETACH); err != nil {
			logger.ErrorF(Name, "Failed to unmount memory partition layer cleanly from layout ERROR: %v", err, err.Error())
		} else {
			logger.DebugAppend(Name, "[REMOVE OVERLAY CLEAN]")
		}
	}

	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if vars.Cmd == nil || vars.Cmd.Process == nil {
		return fmt.Errorf("SAMBA SHARE is not initialized")
	}
	return vars.Cmd.Process.Signal(syscall.Signal(0))
}
