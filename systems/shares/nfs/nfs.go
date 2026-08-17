package nfs

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
	"gorogs/systems/systeminterface"
)

const (
	Name       = "NFS"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

type Struct struct {
	sState     systeminterface.SysStateEnum
	ganeshaCmd *exec.Cmd
	readyChan  chan struct{}
}

func (_ *Struct) Name() string                               { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum       { return Type }
func (_ *Struct) IsCritical() bool                           { return IsCritical }
func (_ *Struct) AutoStart() bool                            { return AutoStart }
func (s *Struct) State(in systeminterface.SysStateEnum) bool { return s.sState == in }

func (s *Struct) Setup() {
	logger.Info(s.Name(), "Commencing pre-flight checks and runtime directory layout parsing...")

	ganeshaPath := "/var/run/ganesha"
	if err := os.MkdirAll(ganeshaPath, 0755); err != nil {
		logger.Fatal(s.Name(), "failed to construct mandatory NFS-Ganesha tracking path", err)
	}

	if err := s.writeGaneshaConfig(); err != nil {
		logger.Fatal(s.Name(), "failed to execute master ganesha config file write utility", err)
	}

	logger.Info(s.Name(), "NFS-Ganesha system runtime configuration generation phase successfully completed.")
	s.sState = systeminterface.SETUP

}

func (s *Struct) writeGaneshaConfig() error {
	configPath := "/etc/ganesha/ganesha.conf"
	logger.Info(s.Name(), "Compiling unified ganesha.conf layout definition parameters...")

	configContent := "NFS_CORE_PARAM {\n" +
		"    Protocols = 3, 4;\n" +
		"    mount_path_pseudo = true;\n" +
		"    Enable_UDP = false;\n" +
		"    NFS_Port = 2049;\n" +
		"    MNT_Port = 892;\n" +
		"    NLM_Port = 4045;\n" +
		"    Rquota_Port = 875;\n" +
		"    Log_File= \"/dev/stderr\"" +
		"}\n\n" +
		"NFSV4 {\n" +
		"    Graceless = true;\n" +
		"}\n\n" +
		"EXPORT {\n" +
		"    Export_Id = 1;\n" +
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

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Spawning containerised user-space NFS-Ganesha storage engine...")

	s.readyChan = make(chan struct{})

	ganeshaArgs := []string{"-F", "-L", "/dev/stdout", "-f", "/etc/ganesha/ganesha.conf"}
	if logger.IsDebugActive(s.Name()) {
		ganeshaArgs = append(ganeshaArgs, "-N", "NIV_FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	ganeshaPipe, err := s.ganeshaCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link Ganesha stdout processing pipeline: %w", err)
	}
	s.ganeshaCmd.Stderr = s.ganeshaCmd.Stdout

	if err := s.ganeshaCmd.Start(); err != nil {
		return fmt.Errorf("failed to initialize background ganesha.nfsd execution loop: %w", err)
	}

	go s.streamSubsystemLogs(ganeshaPipe)

	logger.InfoF(s.Name(), "NFS-Ganesha binary actively supervised under Process ID: %d. Waiting for socket readiness...", s.ganeshaCmd.Process.Pid)

	select {
	case <-s.readyChan:
		logger.Info(s.Name(), "NFS-Ganesha successfully initialized sockets and is accepting connections.")
		s.sState = systeminterface.STARTED
		return nil
	case <-time.After(5 * time.Second):
		return fmt.Errorf("timeout waiting for NFS-Ganesha to declare readiness state milestone")
	}
}

func (s *Struct) streamSubsystemLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	hasSignaledReady := false

	for scanner.Scan() {
		line := scanner.Text()

		if !hasSignaledReady && (strings.Contains(line, "NFS SERVER INITIALIZED") ||
			strings.Contains(line, "General fridge was started successfully")) {

			close(s.readyChan)
			hasSignaledReady = true
		}

		idx := strings.Index(line, "[")
		if idx != -1 {
			line = line[idx:]
		}

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

	if !hasSignaledReady {
		close(s.readyChan)
	}

	if err := scanner.Err(); err != nil {
		logger.Error(s.Name(), "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (s *Struct) Stop() {
	if s.ganeshaCmd != nil && s.ganeshaCmd.Process != nil {
		logger.Info(s.Name(), "Initiating graceful termination sequence on NFS-Ganesha process tree...")
		if err := s.ganeshaCmd.Process.Signal(syscall.SIGTERM); err != nil {
			_ = s.ganeshaCmd.Process.Kill()
		}
		_ = s.ganeshaCmd.Wait()
	}
	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.ganeshaCmd == nil || s.ganeshaCmd.Process == nil {
		return fmt.Errorf("nfs background system execution tracking instance is not initialized")
	}
	return s.ganeshaCmd.Process.Signal(syscall.Signal(0))
}
