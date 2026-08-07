package beacons

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gorogs/config"
	"gorogs/logger"
)

type RpcbindBeacon struct {
	cmd *exec.Cmd
}

func (r *RpcbindBeacon) Setup() error {
	logger.Info("RPCBIND", "Evaluating protocol dependencies and pre-flight requirements...")

	if !config.Instance.NfsEnabled {
		logger.Info("RPCBIND", "NFS storage daemon is disabled. Bypassing dependent RPC portmapper subsystem.")
		return ErrServiceDisabled
	}

	rpcPath := "/run/sendsigs.omit.d"
	if err := os.MkdirAll(rpcPath, 0755); err != nil {
		return fmt.Errorf("failed to construct mandatory rpcbind system tracking directory %s: %w", rpcPath, err)
	}

	logger.Info("RPCBIND", "Subsystem validation check successful. Component ready for boot.")
	return nil
}

func (r *RpcbindBeacon) Start() error {
	logger.Info("RPCBIND", "Spawning background system RPC portmapper daemon...")

	rpcArgs := []string{"-w", "-f"}
	if logger.IsDebugActive("rpcbind") {
		rpcArgs = append(rpcArgs, "-d")
	}

	if !config.Instance.RpcbindEnabled {
		logger.Info("RPCBIND", "RPCBind flag set to disabled. Jailing portmapper socket straight to 127.0.0.1 loopback.")
		rpcArgs = append(rpcArgs, "-h", "127.0.0.1")
	}

	r.cmd = exec.Command("/usr/sbin/rpcbind", rpcArgs...)
	r.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	rpcStdout, err := r.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stdout pipe: %w", err)
	}
	rpcStderr, err := r.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stderr pipe: %w", err)
	}

	if err := r.cmd.Start(); err != nil {
		return fmt.Errorf("failed to execute local rpcbind utility loop: %w", err)
	}

	go r.streamRpcbindLogs(rpcStdout)
	go r.streamRpcbindLogs(rpcStderr)

	logger.Info("RPCBIND", fmt.Sprintf("RPC portmapper tracking loop active under process ID: %d", r.cmd.Process.Pid))
	return nil
}

func (r *RpcbindBeacon) streamRpcbindLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if logger.IsDebugActive("rpcbind") {
			logger.Debug("RPCBIND", trimmedLine)
		} else {
			logger.Info("RPCBIND", trimmedLine)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("RPCBIND", "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (r *RpcbindBeacon) Healthcheck() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return fmt.Errorf("rpcbind background daemon execution instance is uninitialized")
	}
	return r.cmd.Process.Signal(syscall.Signal(0))
}

func (r *RpcbindBeacon) IsCritical() bool {
	return false
}

func (r *RpcbindBeacon) Stop() error {
	if r.cmd == nil || r.cmd.Process == nil {
		return nil
	}

	logger.Info("RPCBIND", "Conveying termination signal to system RPC daemon threads...")
	if err := r.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return r.cmd.Process.Kill()
	}

	_ = r.cmd.Wait()
	return nil
}
