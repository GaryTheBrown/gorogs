package shares

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"gorogs/app/config"
	"gorogs/app/logger"
)

type NfsShare struct {
	rpcCmd     *exec.Cmd
	ganeshaCmd *exec.Cmd
}

func (n *NfsShare) Setup() error {
	logger.Info("NFS", "Commencing pre-flight checks and runtime directory layout parsing...")

	ganeshaPath := "/var/run/ganesha"
	if err := os.MkdirAll(ganeshaPath, 0755); err != nil {
		return fmt.Errorf("failed to construct mandatory NFS-Ganesha tracking path: %w", err)
	}

	rpcPath := "/run/sendsigs.omit.d"
	if err := os.MkdirAll(rpcPath, 0755); err != nil {
		return fmt.Errorf("failed to construct mandatory rpcbind system tracking directory: %w", err)
	}

	if err := n.writeGaneshaConfig(); err != nil {
		return fmt.Errorf("failed to execute master ganesha config file write utility: %w", err)
	}

	logger.Info("NFS", "NFS-Ganesha system runtime configuration generation phase successfully completed.")
	return nil
}

func (n *NfsShare) writeGaneshaConfig() error {
	configPath := "/etc/ganesha/ganesha.conf"
	logger.Info("NFS", "Compiling unified ganesha.conf layout definition parameters...")

	configContent := "NFS_CORE_PARAM {\n" +
		"    Protocols = 3, 4;\n" +
		"    mount_path_pseudo = true;\n" +
		"    Enable_UDP = false;\n" +
		"    NFS_Port = 2049;\n" +
		"    MNT_Port = 892;\n" +
		"    NLM_Port = 4045;\n" +
		"    Rquota_Port = 875;\n" +
		"}\n\n" +
		"EXPORT {\n" +
		"    Export_Id = 0;\n" +
		"    Path = " + config.ShareRoot + ";\n" +
		"    Pseudo = /;\n" +
		"    Access_Type = RO;\n" +
		"    Protocols = 3, 4;\n" +
		"    SecType = \"sys\";\n" +
		"    FSAL {\n" +
		"        Name = VFS;\n" +
		"    }\n" +
		"}\n"

	return os.WriteFile(configPath, []byte(configContent), 0644)
}

func (n *NfsShare) Start() error {
	logger.Info("NFS", "Spawning background system RPC portmapper daemon...")

	rpcArgs := []string{"-w"}
	if logger.IsDebugActive("rpcbind") {
		rpcArgs = append(rpcArgs, "-d")
	}

	n.rpcCmd = exec.Command("/usr/sbin/rpcbind", rpcArgs...)
	n.rpcCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture both stdout and stderr pipes from rpcbind to capture all logs
	rpcStdout, err := n.rpcCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stdout pipe: %w", err)
	}
	rpcStderr, err := n.rpcCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to link RPC stderr pipe: %w", err)
	}

	if err := n.rpcCmd.Start(); err != nil {
		return fmt.Errorf("failed to launch background rpcbind daemon: %w", err)
	}

	// Scan both stdout and stderr asynchronously for rpcbind logs
	go n.streamSubsystemLogs("RPCBIND", rpcStdout)
	go n.streamSubsystemLogs("RPCBIND", rpcStderr)

	time.Sleep(200 * time.Millisecond)

	logger.Info("NFS", "Spawning containerised user-space NFS-Ganesha storage engine...")

	ganeshaArgs := []string{"-F", "-L", "/dev/stdout", "-f", "/etc/ganesha/ganesha.conf"}
	if logger.IsDebugActive("nfs") {
		ganeshaArgs = append(ganeshaArgs, "-N", "NIV_FULL_DEBUG")
	}

	n.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	n.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	ganeshaPipe, err := n.ganeshaCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link Ganesha stdout processing pipeline: %w", err)
	}
	n.ganeshaCmd.Stderr = n.ganeshaCmd.Stdout

	if err := n.ganeshaCmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background ganesha.nfsd execution loop: %w", err)
	}

	go n.streamSubsystemLogs("NFS", ganeshaPipe)

	logger.Info("NFS", fmt.Sprintf("NFS-Ganesha binary actively supervised under Process ID: %d", n.ganeshaCmd.Process.Pid))
	return nil
}

func (n *NfsShare) streamSubsystemLogs(subsystem string, pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()

		if subsystem == "NFS" {
			idx := strings.Index(line, "[")
			if idx != -1 {
				line = line[idx:]
			}
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if logger.IsDebugActive(strings.ToLower(subsystem)) {
			logger.Debug(subsystem, trimmedLine)
		} else {
			logger.Info(subsystem, trimmedLine)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error(subsystem, "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (n *NfsShare) Healthcheck() error {
	if n.rpcCmd == nil || n.rpcCmd.Process == nil {
		return fmt.Errorf("rpcbind background daemon tracking instance is uninitialized")
	}
	if err := n.rpcCmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("rpcbind background daemon has stalled or terminated unexpectedly: %w", err)
	}

	if n.ganeshaCmd == nil || n.ganeshaCmd.Process == nil {
		return fmt.Errorf("nfs background system execution tracking instance is not initialized")
	}
	if err := n.ganeshaCmd.Process.Signal(syscall.Signal(0)); err != nil {
		return fmt.Errorf("nfs background process loop has hung or terminated: %w", err)
	}

	return nil
}

func (n *NfsShare) IsCritical() bool { return true }

func (n *NfsShare) Stop() error {
	logger.Info("NFS", "Initiating graceful termination sequence on NFS-Ganesha and RPC process trees...")

	if n.ganeshaCmd != nil && n.ganeshaCmd.Process != nil {
		if err := n.ganeshaCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = n.ganeshaCmd.Process.Kill()
		}
		_ = n.ganeshaCmd.Wait()
	}

	if n.rpcCmd != nil && n.rpcCmd.Process != nil {
		if err := n.rpcCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = n.rpcCmd.Process.Kill()
		}
		_ = n.rpcCmd.Wait()
	}

	return nil
}
