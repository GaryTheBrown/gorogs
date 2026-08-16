package netbios

import (
	"bufio"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type NetBIOSStruct struct {
	sState systems.SysStateEnum
	cmd    *exec.Cmd
}

func (_ NetBIOSStruct) Name() string                        { return "nmbd" }
func (_ NetBIOSStruct) Type() systems.SystemTypeEnum        { return systems.Beacon }
func (_ NetBIOSStruct) IsCritical() bool                    { return false }
func (_ NetBIOSStruct) AutoStart() bool                     { return false }
func (n *NetBIOSStruct) State(in systems.SysStateEnum) bool { return n.sState == in }

func (n *NetBIOSStruct) Setup() error {
	logger.Info("NMBD", "Commencing pre-flight checks and isolated NetBIOS configuration generation...")

	if err := os.MkdirAll("/var/log/samba", 0755); err != nil {
		return fmt.Errorf("failed to provision local samba logging directories: %w", err)
	}

	if err := n.writeMasterNmbdConfig(config.Hostname); err != nil {
		return fmt.Errorf("failed to execute master nmbd config write utility: %w", err)
	}

	logger.Info("NMBD", "NetBIOS beacon configuration generation phase successfully completed.")
	n.sState = systems.SETUP
	return nil
}

func (n *NetBIOSStruct) writeMasterNmbdConfig(serverName string) error {
	masterConfigPath := "/etc/samba/nmbd.conf"

	masterContent := "[global]\n" +
		"    netbios name = " + serverName + "\n" +
		"    workgroup = WORKGROUP\n" +
		"    server string = Network Discovery Beacon\n" +
		"    log file = /var/log/samba/log.nmbd\n" +
		"    max log size = 1000\n" +
		"    logging = file\n" +
		"    local master = no\n" +
		"    preferred master = no\n" +
		"    domain master = no\n"

	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

func (n *NetBIOSStruct) Start() error {
	logger.Info("NMBD", "Spawning standalone NetBIOS name discovery beacon companion (nmbd)...")

	n.cmd = exec.Command("/usr/sbin/nmbd", "--foreground", "--no-process-group", "--debug-stdout", "-s", "/etc/samba/nmbd.conf")
	n.cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	nmbPipe, err := n.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to link NetBIOS nmbd stdout processing pipeline: %w", err)
	}
	n.cmd.Stderr = n.cmd.Stdout

	if err := n.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start standalone nmbd beacon process: %w", err)
	}

	go n.streamSubsystemLogs(nmbPipe)

	logger.InfoF("NMBD", "NetBIOS tracker process running (PID: %d). Probing UDP socket...", n.cmd.Process.Pid)

	for i := 0; i < 30; i++ {
		conn, err := net.DialTimeout("udp4", "127.0.0.1:137", 500*time.Millisecond)
		if err == nil {
			conn.Close()
			logger.Info("NMBD", "Verified: NetBIOS daemon has successfully bound UDP Port 137 and is online.")
			n.sState = systems.STARTED
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("timeout waiting for NetBIOS network interface to bind socket 137")
}

func (n *NetBIOSStruct) streamSubsystemLogs(pipe io.ReadCloser) {
	defer pipe.Close()
	scanner := bufio.NewScanner(pipe)

	for scanner.Scan() {
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}
		logger.Info("NMBD", trimmedLine)
	}

	if err := scanner.Err(); err != nil {
		logger.Error("NMBD", "Log scanning utility loop encountered an underlying stream parsing error", err)
	}
}

func (n *NetBIOSStruct) Healthcheck() error {
	if n.cmd == nil || n.cmd.Process == nil {
		return fmt.Errorf("netbios background system execution tracking instance is not initialized")
	}
	return n.cmd.Process.Signal(syscall.Signal(0))
}

func (n *NetBIOSStruct) Stop() error {
	if n.cmd == nil || n.cmd.Process == nil {
		return nil
	}
	logger.Info("NMBD", "Initiating graceful termination sequence on NetBIOS beacon threads...")
	if err := n.cmd.Process.Signal(syscall.SIGTERM); err != nil {
		return n.cmd.Process.Kill()
	}
	_ = n.cmd.Wait()
	n.sState = systems.STOPPED
	return nil
}
