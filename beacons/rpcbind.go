package beacons

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

	// 1. Create the system termination tracking directory
	rpcPath := "/run/sendsigs.omit.d"
	if err := os.MkdirAll(rpcPath, 0755); err != nil {
		return fmt.Errorf("failed to construct mandatory rpcbind system tracking directory %s: %w", rpcPath, err)
	}

	// 2. Ensure the mandatory socket directory exists for modern rpcbind binaries
	runRpcbindPath := "/run/rpcbind"
	if err := os.MkdirAll(runRpcbindPath, 0755); err != nil {
		return fmt.Errorf("failed to construct essential runtime socket directory %s: %w", runRpcbindPath, err)
	}

	// 3. Verify or inject basic rpcbind protocol ports to prevent resolution drops
	servicesPath := "/etc/services"
	if _, err := os.Stat(servicesPath); os.IsNotExist(err) {
		logger.Info("RPCBIND", "Notice: System /etc/services layout missing. Compiling fallback rules...")
		fallbackServices := "sunrpc          111/tcp         portmapper rpcbind\n" +
			"sunrpc          111/udp         portmapper rpcbind\n"
		_ = os.WriteFile(servicesPath, []byte(fallbackServices), 0644)
	}
	// =========================================================================

	logger.Info("RPCBIND", "Subsystem validation check successful. Component ready for boot.")
	return nil
}

func (r *RpcbindBeacon) Start() error {
	logger.Info("RPCBIND", "Spawning background system RPC portmapper daemon...")

	rpcArgs := []string{"-w", "-f"}
	if logger.IsDebugActive("rpcbind") {
		rpcArgs = append(rpcArgs, "-d")
	}

	containerIPStr := config.Instance.ContainerIP.String()

	if !config.Instance.RpcbindEnabled {
		logger.Info("RPCBIND", "RPCBind flag set to disabled. Binding portmapper explicitly to container IP layout.")
		rpcArgs = append(rpcArgs, "-h", containerIPStr)
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

	// --- FINAL BULLETPROOF TIMING CHECK ---
	dialTarget := "127.0.0.1:111"
	logger.Info("RPCBIND", "Verifying portmapper socket readiness on loopback channel...")

	rpcReady := false
	for i := 0; i < 10; i++ { // Check for up to 5 seconds total (10 * 500ms)

		// Diagnostic step: Verify the binary process didn't die immediately after Start()
		if r.cmd.ProcessState != nil && r.cmd.ProcessState.Exited() {
			return fmt.Errorf("rpcbind daemon process terminated prematurely with exit code status")
		}

		conn, err := net.DialTimeout("tcp", dialTarget, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			rpcReady = true
			logger.Info("RPCBIND", "RPC portmapper socket successfully initialized and synchronized.")
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	if !rpcReady {
		return fmt.Errorf("timeout waiting for rpcbind process to open port 111")
	}
	// --------------------------------------

	logger.Info("RPCBIND", fmt.Sprintf("RPC portmapper tracking loop active and listening under process ID: %d", r.cmd.Process.Pid))
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
