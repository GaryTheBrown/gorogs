package nfs

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
	Name       = "NFS"
	Type       = systeminterface.Share
	IsCritical = true
	AutoStart  = true
)

var (
	programPath = "/bin/ganesha.nfsd"
	ganeshaConf = "/etc/ganesha.conf"
)

type Struct struct {
	sState     systeminterface.SysStateEnum
	ganeshaCmd *exec.Cmd
	logWriter  *helpers.SubsystemWriter
	readyChan  chan struct{}
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

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
	configPath := "/etc/ganesha.conf"
	logger.Info(s.Name(), "Compiling unified ganesha.conf layout definition parameters...")

	// Extract physical ownership metrics dynamically from the target path directory metadata
	uid := uint32(1000)
	gid := uint32(1000)
	fi, err := os.Stat(config.ShareRoot)
	if err != nil {
		logger.Warn(s.Name(), fmt.Sprintf("Unable to stat ShareRoot path '%s', defaulting anonymous UID/GID mapping to 1000: %v", config.ShareRoot, err))
	} else {
		if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
			uid = stat.Uid
			gid = stat.Gid
		}
	}

	gLogLevel := "EVENT"
	if logger.IsDebugActive(Name) {
		gLogLevel = "DEBUG"
	}

	configContent := "NFS_CORE_PARAM {\n" +
		"    NFS_Protocols = 3, 4;\n" +
		"    mount_path_pseudo = true;\n" +
		"    NFS_Port = 2049;\n" +
		"    MNT_Port = 20048;\n" +
		"    NLM_Port = 32803;\n" +
		"    Rquota_Port = 875;\n" +
		"}\n\n" +
		"LOG {\n" +
		"    Default_Log_Level = " + gLogLevel + ";\n" +
		"}\n\n" +
		"NFSV4 {\n" +
		"    Graceless = true;\n" +
		"    DomainName = \"" + config.DomainName + "\";\n" +
		"}\n\n" +

		"EXPORT {\n" +
		"    Export_Id = 1;\n" +
		"    Path = " + config.ShareRoot + ";\n" +
		"    Pseudo = /;\n" +
		"    Access_Type = RO;\n" +
		"    Protocols = 3, 4;\n" +
		"    SecType = \"sys\";\n" +
		"    Squash = All_Squash;\n" +
		fmt.Sprintf("    Anonymous_Uid = %d;\n", uid) +
		fmt.Sprintf("    Anonymous_Gid = %d;\n", gid) +
		"    FSAL {\n" +
		"        Name = VFS;\n" +
		"    }\n" +
		"}\n"

	return os.WriteFile(configPath, []byte(configContent), 0644)
}

func (s *Struct) Start() error {
	logger.Info(s.Name(), "Spawning containerised user-space NFS-Ganesha storage engine...")

	s.readyChan = make(chan struct{})
	ganeshaArgs := []string{"-F", "-f", ganeshaConf}

	if logger.IsDebugActive(s.Name()) {
		ganeshaArgs = append(ganeshaArgs, "-L", "/dev/stdout", "-N", "NIV_FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	if logger.IsDebugActive(s.Name()) {
		phrases := []string{"NFS SERVER INITIALIZED", "General fridge was started successfully"}
		s.logWriter = helpers.NewSubsystemWriter(s.Name(), s.readyChan, phrases, nil)
		s.ganeshaCmd.Stdout = s.logWriter
		s.ganeshaCmd.Stderr = s.logWriter
	} else {
		s.logWriter = helpers.NewSubsystemWriter(s.Name(), nil, nil, nil)
		s.ganeshaCmd.Stdout = s.logWriter
		s.ganeshaCmd.Stderr = s.logWriter
	}

	if err := s.ganeshaCmd.Start(); err != nil {
		close(s.readyChan)
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to initialize background ganesha.nfsd: %w", err)
	}

	logger.InfoF(s.Name(), "NFS-Ganesha binary actively supervised under Process ID: %d. Synchronizing socket state...", s.ganeshaCmd.Process.Pid)

	if logger.IsDebugActive(s.Name()) {
		select {
		case <-s.readyChan:
			logger.Info(s.Name(), "NFS-Ganesha successfully verified milestone log signatures and is online.")
			s.sState = systeminterface.STARTED
			return nil
		case <-time.After(10 * time.Second):
			if s.logWriter != nil {
				_ = s.logWriter.Close()
			}
			return fmt.Errorf("timeout waiting for NFS-Ganesha to declare readiness state milestone log tags")
		}
	} else {
		if !helpers.WaitForSocket("tcp", "127.0.0.1:2049", 10*time.Second) {
			if s.logWriter != nil {
				_ = s.logWriter.Close()
			}
			return fmt.Errorf("timeout waiting for production NFS-Ganesha daemon to bind port 2049")
		}
		logger.Info(s.Name(), "NFS-Ganesha network port 2049 successfully bound and listening.")
		s.sState = systeminterface.STARTED
		return nil
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

	if s.logWriter != nil {
		_ = s.logWriter.Close()
	}

	s.sState = systeminterface.STOPPED
}

func (s *Struct) Healthcheck() error {
	if s.ganeshaCmd == nil || s.ganeshaCmd.Process == nil {
		return fmt.Errorf("nfs background system execution tracking instance is not initialized")
	}
	return s.ganeshaCmd.Process.Signal(syscall.Signal(0))
}
