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
	programPath = "/usr/bin/ganesha.nfsd"
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

func (s *Struct) Config() {

}

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "System Setup...")

	if err := s.writeGaneshaConfig(); err != nil {
		logger.Fatal(s.Name(), "failed to execute master ganesha config file write utility", err)
	}
	logger.DebugAppend(Name, "[write config]")

	s.sState = systeminterface.SETUP
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) writeGaneshaConfig() error {
	configPath := "/etc/ganesha.conf"

	gLogLevel := "EVENT"
	if logger.IsDebugActive(Name) {
		gLogLevel = "DEBUG"
	}

	configContent := fmt.Sprintf(`NFS_CORE_PARAM {
    NFS_Protocols = 3, 4;
    mount_path_pseudo = true;
    NFS_Port = 2049;
    MNT_Port = 20048;
    NLM_Port = 32803;
    Rquota_Port = 875;
}

LOG {
    Default_Log_Level = %s;
}

NFSV4 {
    Graceless = true;
    DomainName = "%s";
}

EXPORT {
    Export_Id = 1;
    Path = %s;
    Pseudo = /;
    Access_Type = RO;
    Protocols = 3, 4;
    SecType = "sys";
    Squash = All_Squash;
    Anonymous_Uid = 65534;
    Anonymous_Gid = 65534;
    FSAL {
        Name = VFS;
        # Prevents client "Stale File Handles" if gorogs rewrites or reloads configs
        Filesystem_Id = 1.1;
    }
}`, gLogLevel, config.DomainName, config.ShareRoot)

	return os.WriteFile(configPath, []byte(configContent), 0644)
}

func (s *Struct) Start() error {
	logger.DebugContinue(Name, "System Starting...")
	if err := s.StartProgram(); err != nil {
		return err
	}
	if err := s.Wait(); err != nil {
		return err
	}
	s.sState = systeminterface.STARTED
	logger.DebugEnd(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	if s.ganeshaCmd != nil && s.ganeshaCmd.Process != nil {
		logger.DebugContinue(Name, "Stopping NFS daemon threads...")

		_ = s.ganeshaCmd.Process.Signal(syscall.SIGTERM)

		go func(p *os.Process) {
			time.Sleep(150 * time.Millisecond)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				logger.Debug(s.Name(), "NFS daemon socket locked. Enforcing asynchronous SIGKILL override pass.")
				_ = p.Kill()
			}
		}(s.ganeshaCmd.Process)

		_ = s.ganeshaCmd.Wait()
		logger.DebugAppend(Name, "[CMD Stop]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.DebugAppend(Name, "[STDOUT->LOG STOP]")
	}

	s.sState = systeminterface.STOPPED
	logger.DebugEnd(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.ganeshaCmd == nil || s.ganeshaCmd.Process == nil {
		return fmt.Errorf("NFS SHARE is not initialized")
	}
	return s.ganeshaCmd.Process.Signal(syscall.Signal(0))
}
