package rpcbind

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "RPCBind"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

var (
	programPath = "/usr/bin/rpcbind"
	statdPath   = "/usr/bin/rpc.statd"
)

type Struct struct {
	sState      systeminterface.SysStateEnum
	rpcCmd      *exec.Cmd
	statdCmd    *exec.Cmd
	rpcWriter   *helpers.SubsystemWriter
	statdWriter *helpers.SubsystemWriter
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup() {
	logger.Info(s.Name(), "Evaluating protocol dependencies and pre-flight requirements...")
	rpcPath := "/run/sendsigs.omit.d"
	if err := os.MkdirAll(rpcPath, 0755); err != nil {
		logger.FatalF(s.Name(), "failed to construct mandatory rpcbind system tracking directory %s: %w", err, rpcPath)
	}

	runRpcbindPath := "/run/rpcbind"
	if err := os.MkdirAll(runRpcbindPath, 0755); err != nil {
		logger.FatalF(s.Name(), "failed to construct essential runtime socket directory %s: %w", err, runRpcbindPath)
	}

	servicesPath := "/etc/services"
	if _, err := os.Stat(servicesPath); os.IsNotExist(err) {
		logger.Info(s.Name(), "Notice: System /etc/services layout missing. Compiling fallback rules...")
		fallbackServices := "sunrpc          111/tcp         portmapper rpcbind\n" +
			"sunrpc          111/udp         portmapper rpcbind\n"
		_ = os.WriteFile(servicesPath, []byte(fallbackServices), 0644)
	}

	logger.Info(s.Name(), "Subsystem validation check successful. Component ready for boot.")
	s.sState = systeminterface.SETUP
}

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Spawning background system RPC portmapper daemon...")

	rpcArgs := []string{"-w", "-f"}
	if logger.IsDebugActive(s.Name()) {
		rpcArgs = append(rpcArgs, "-d")
	}

	containerIPStr := config.SystemIP.String()

	if config.IsDisabled(s.Name()) {
		logger.Info(s.Name(), "RPCBind flag set to disabled. Binding portmapper explicitly to container IP layout.")
		rpcArgs = append(rpcArgs, "-h", containerIPStr)
	}

	s.rpcCmd = exec.Command(programPath, rpcArgs...)
	s.rpcCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if logger.IsDebugActive(s.Name()) {
		s.rpcWriter = helpers.NewSubsystemWriter(s.Name(), nil, nil, nil)
		s.rpcCmd.Stdout = s.rpcWriter
		s.rpcCmd.Stderr = s.rpcWriter
	}

	if err := s.rpcCmd.Start(); err != nil {
		if s.rpcWriter != nil {
			_ = s.rpcWriter.Close()
		}
		return fmt.Errorf("failed to execute local rpcbind utility loop: %w", err)
	}

	logger.Info(s.Name(), "Verifying portmapper socket readiness on loopback channel...")

	if !helpers.WaitForSocket("tcp", "127.0.0.1:111", 5*time.Second) {
		if s.rpcWriter != nil {
			_ = s.rpcWriter.Close()
		}
		return fmt.Errorf("timeout waiting for rpcbind process to open port 111 or process exited early")
	}
	logger.Info(s.Name(), "RPC portmapper socket successfully initialized and synchronized.")

	logger.Info(s.Name(), "Spawning background NFSv3 status monitor daemon (rpc.statd)...")

	_ = os.MkdirAll("/var/lib/nfs/sm", 0755)
	_ = os.MkdirAll("/var/lib/nfs/sm.bak", 0755)

	s.statdCmd = exec.Command(statdPath, "-F")
	s.statdCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if logger.IsDebugActive(s.Name()) {
		s.statdWriter = helpers.NewSubsystemWriter(s.Name(), nil, nil, nil)
		s.statdCmd.Stdout = s.statdWriter
		s.statdCmd.Stderr = s.statdWriter
	}

	if err := s.statdCmd.Start(); err != nil {
		logger.Error(s.Name(), "Failed to launch network status monitor process tree", err)
		if s.statdWriter != nil {
			_ = s.statdWriter.Close()
		}
	} else {
		logger.InfoF(s.Name(), "NFSv3 statd tool active under process ID: %d", s.statdCmd.Process.Pid)
	}

	logger.InfoF(s.Name(), "RPC portmapper tracking loop active and listening under process ID: %d", s.rpcCmd.Process.Pid)
	s.sState = systeminterface.STARTED
	return nil
}

func (s *Struct) Stop() {
	if s.statdCmd != nil && s.statdCmd.Process != nil {
		logger.Info(s.Name(), "Conveying termination signal to system statd threads...")
		if err := s.statdCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.statdCmd.Process.Kill()
		}
		_ = s.statdCmd.Wait()
	}

	if s.statdWriter != nil {
		_ = s.statdWriter.Close()
	}

	if s.rpcCmd != nil && s.rpcCmd.Process != nil {
		logger.Info(s.Name(), "Conveying termination signal to system RPC daemon threads...")
		if err := s.rpcCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.rpcCmd.Process.Kill()
		}
		_ = s.rpcCmd.Wait()
	}

	if s.rpcWriter != nil {
		_ = s.rpcWriter.Close()
	}

	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.rpcCmd == nil || s.rpcCmd.Process == nil {
		return fmt.Errorf("rpcbind background daemon execution instance is uninitialized")
	}
	return s.rpcCmd.Process.Signal(syscall.Signal(0))
}
