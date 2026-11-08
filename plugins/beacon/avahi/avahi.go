package main

import (
	"fmt"
	"gorogs/config"
	"gorogs/helpers"
	"gorogs/logger"
	"gorogs/system"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	Name       string                = "avahi"
	Type       system.SystemTypeEnum = system.Beacon
	IsCritical bool                  = false
	AutoStart  bool                  = true
)

var (
	programPath       string = "/usr/bin/avahi-daemon"
	masterConfigPath  string = "/etc/avahi/avahi-daemon.conf"
	serviceConfigPath string = "/etc/avahi/services"
	deviceInfoService string = serviceConfigPath + "/deviceinfo.service"
	sambaService      string = serviceConfigPath + "/smb.service"
	nfsService        string = serviceConfigPath + "/nfs.service"
)

type Struct struct {
	sState    system.SysStateEnum
	cmd       *exec.Cmd
	logWriter *config.SubsystemWriter

	disabled map[string]bool
}

func (_ *Struct) Name() string                        { return Name }
func (_ *Struct) Type() system.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                    { return IsCritical }
func (_ *Struct) AutoStart() bool                     { return AutoStart }
func (s *Struct) IsState(in system.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() system.SysStateEnum       { return s.sState }
func (_ *Struct) Dependencies() []string              { return nil }
func (_ *Struct) OrderAfter() []string                { return []string{"nfs", "samba"} }
func (_ *Struct) Priority() int                       { return 100 }

var SystemInstance Struct

func init() {
	system.Register(&SystemInstance)
}

func (s *Struct) Config(cm config.ConfigMap) {
	s.disabled = make(map[string]bool)
	disabledStr := cm.Get("disabled", "")
	disabledList := strings.SplitSeq(strings.ToLower(disabledStr), ",")
	for di := range disabledList {
		s.disabled[di] = true
	}

	services["device-info"].TxtRecords["model"] = cm.Get("servericon", "TimeCapsule8,119")
}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")

	if err := os.MkdirAll(serviceConfigPath, 0755); err != nil {
		logger.FatalF(Name, "failed to initialize system directory footprint surfaces: %v", err)
	}
	logger.Debug(Name, "[directory path tree initialized]")

	if err := s.writeMasterConfig(); err != nil {
		logger.FatalF(Name, "failed to write %s config file", err, Name)
	}
	logger.Debug(Name, "[write config]")
	if err := s.writeServiceConfig(); err != nil {
		logger.FatalF(Name, "failed to write %s services map: %w", err, Name)
	}
	logger.Debug(Name, "[write services map]")

	s.sState = system.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) writeMasterConfig() error {
	masterContent := fmt.Sprintf(`[server]
host-name=%s
domain-name=local
use-ipv4=yes
use-ipv6=no
enable-dbus=no
allow-interfaces=%s
deny-interfaces=lo
use-iff-running=yes
disallow-other-stacks=yes

[wide-area]
enable-wide-area=yes

[publish]
add-service-cookie=no
publish-addresses=yes
publish-hinfo=no
publish-workstation=yes
publish-domain=yes

[reflector]
enable-reflector=yes`,
		config.Hostname,
		strings.Join(config.NetworkInterfaces, ","),
	)
	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (s *Struct) writeServiceConfig() error {
	_, deviceInfoDisabled := s.disabled["device-info"]
	if !deviceInfoDisabled {
		if sd, ok := services["device-info"]; ok {
			b := sd.Service()
			logger.Info(Name, string(b))
			err := os.WriteFile(deviceInfoService, b, 0644)
			if err != nil {
				return err
			}
		}
	}
	_, sambaDisabled := s.disabled["samba"]
	if !sambaDisabled {
		if sd, ok := services["samba"]; ok {
			b := sd.Service()
			logger.Info(Name, string(b))
			err := os.WriteFile(sambaService, b, 0644)
			if err != nil {
				return err
			}
		}
	}
	_, nfsDisabled := s.disabled["nfs"]
	if !nfsDisabled {
		if sd, ok := services["nfs"]; ok {
			b := sd.Service()
			logger.Info(Name, string(b))
			err := os.WriteFile(nfsService, b, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")

	s.cmd = exec.Command(programPath, "--no-drop-root")
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	s.logWriter = config.NewSubsystemWriter(Name, nil)
	s.cmd.Stdout = s.logWriter
	s.cmd.Stderr = s.logWriter
	logger.Debug(Name, "[LINK STDOUT->LOG]")

	if err := s.cmd.Start(); err != nil {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to start standalone %s beacon process: %w", Name, err)
	}
	logger.Debug(Name, "[CMD START]")

	if !helpers.WaitForSocket("udp4", "127.0.0.1:5353", 15*time.Second) {
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		logger.Debug(Name, "[TIMEOUT][FAILED]")
		return fmt.Errorf("timeout waiting for %s network interface to bind socket 5353", Name)
	}

	s.sState = system.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	logger.DebugF(Name, "Stopping %s...", Name)
	if s.cmd != nil && s.cmd.Process != nil {

		if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.cmd.Process.Kill()
		}

		_ = s.cmd.Wait()
		logger.Debug(Name, "[CMD Stop]")
	}

	_ = os.Remove(masterConfigPath)
	_ = os.Remove(deviceInfoService)
	_ = os.Remove(sambaService)
	_ = os.Remove(nfsService)

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.Debug(Name, "[STDOUT->LOG STOP]")
	}
	s.sState = system.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("%s is not initialized", Name)
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
