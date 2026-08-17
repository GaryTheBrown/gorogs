package rpcbind

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "RPCBind"
	Type       = systeminterface.Beacon
	IsCritical = false
	AutoStart  = true
)

type Struct struct {
	sState   systeminterface.SysStateEnum
	cmd      *exec.Cmd
	statdCmd *exec.Cmd
}

func (_ *Struct) Name() string                               { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum       { return Type }
func (_ *Struct) IsCritical() bool                           { return IsCritical }
func (_ *Struct) AutoStart() bool                            { return AutoStart }
func (s *Struct) State(in systeminterface.SysStateEnum) bool { return s.sState == in }

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

	s.cmd = exec.Command("/usr/sbin/rpcbind", rpcArgs...)
	s.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	rpcStdout, err := s.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stdout pipe: %w", err)
	}
	rpcStderr, err := s.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stderr pipe: %w", err)
	}

	if err := s.cmd.Start(); err != nil {
		return fmt.Errorf("failed to execute local rpcbind utility loop: %w", err)
	}
	go s.streamRpcbindLogs(rpcStdout)
	go s.streamRpcbindLogs(rpcStderr)

	dialTarget := "127.0.0.1:111"
	logger.Info(s.Name(), "Verifying portmapper socket readiness on loopback channel...")

	rpcReady := false
	for i := 0; i < 10; i++ {

		if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
			return fmt.Errorf("rpcbind daemon process terminated prematurely with exit code status")
		}

		conn, err := net.DialTimeout("tcp", dialTarget, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			rpcReady = true
			logger.Info(s.Name(), "RPC portmapper socket successfully initialized and synchronized.")
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	if !rpcReady {
		return fmt.Errorf("timeout waiting for rpcbind process to open port 111")
	}

	logger.Info(s.Name(), "Spawning background NFSv3 status monitor daemon (rpc.statd)...")

	_ = os.MkdirAll("/var/lib/nfs/sm", 0755)
	_ = os.MkdirAll("/var/lib/nfs/sm.bak", 0755)

	s.statdCmd = exec.Command("/usr/sbin/rpc.statd", "-F")
	s.statdCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	statdStdout, _ := s.statdCmd.StdoutPipe()
	statdStderr, _ := s.statdCmd.StderrPipe()

	if err := s.statdCmd.Start(); err != nil {
		logger.Error(s.Name(), "Failed to launch network status monitor process tree", err)
	} else {
		go s.streamRpcbindLogs(statdStdout)
		go s.streamRpcbindLogs(statdStderr)
		logger.InfoF(s.Name(), "NFSv3 statd tool active under process ID: %d", s.statdCmd.Process.Pid)
	}

	logger.InfoF(s.Name(), "RPC portmapper tracking loop active and listening under process ID: %d", s.cmd.Process.Pid)
	s.sState = systeminterface.STARTED
	return nil
}

func (s *Struct) streamRpcbindLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if logger.IsDebugActive(s.Name()) {
			logger.Debug(s.Name(), trimmedLine)
		} else {
			logger.Info(s.Name(), trimmedLine)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error(s.Name(), "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (s *Struct) Stop() {
	if s.statdCmd != nil && s.statdCmd.Process != nil {
		logger.Info(s.Name(), "Conveying termination signal to system statd threads...")
		_ = s.statdCmd.Process.Signal(syscall.SIGTERM)
		_ = s.statdCmd.Wait()
	}

	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	logger.Info(s.Name(), "Conveying termination signal to system RPC daemon threads...")
	if err := s.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		s.cmd.Process.Kill()
		return
	}

	_ = s.cmd.Wait()
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.cmd == nil || s.cmd.Process == nil {
		return fmt.Errorf("rpcbind background daemon execution instance is uninitialized")
	}
	return s.cmd.Process.Signal(syscall.Signal(0))
}
