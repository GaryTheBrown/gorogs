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

	"gorogs/config"
	"gorogs/logger"
)

type NfsShare struct {
	ganeshaCmd *exec.Cmd
}

func (n *NfsShare) Setup() error {
	logger.Info("NFS", "Commencing pre-flight checks and runtime directory layout parsing...")

	ganeshaPath := "/var/run/ganesha"
	if err := os.MkdirAll(ganeshaPath, 0755); err != nil {
		return fmt.Errorf("failed to construct mandatory NFS-Ganesha tracking path: %w", err)
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

	go n.streamSubsystemLogs(ganeshaPipe)

	logger.Info("NFS", fmt.Sprintf("NFS-Ganesha binary actively supervised under Process ID: %d", n.ganeshaCmd.Process.Pid))
	return nil
}

func (n *NfsShare) streamSubsystemLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()

		idx := strings.Index(line, "[")
		if idx != -1 {
			line = line[idx:]
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		if logger.IsDebugActive("nfs") {
			logger.Debug("NFS", trimmedLine)
		} else {
			logger.Info("NFS", trimmedLine)
		}
	}

	if err := scanner.Err(); err != nil {
		logger.Error("NFS", "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (n *NfsShare) Healthcheck() error {
	if n.ganeshaCmd == nil || n.ganeshaCmd.Process == nil {
		return fmt.Errorf("nfs background system execution tracking instance is not initialized")
	}
	return n.ganeshaCmd.Process.Signal(syscall.Signal(0))
}

func (n *NfsShare) IsCritical() bool { return true }

func (n *NfsShare) Stop() error {
	if n.ganeshaCmd != nil && n.ganeshaCmd.Process != nil {
		logger.Info("NFS", "Initiating graceful termination sequence on NFS-Ganesha process tree...")
		if err := n.ganeshaCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = n.ganeshaCmd.Process.Kill()
		}
		_ = n.ganeshaCmd.Wait()
	}
	return nil
}
