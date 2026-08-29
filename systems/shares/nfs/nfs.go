package nfs

import (
	"fmt"
	"net"
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

const CONFIGFILE = `NFS_CORE_PARAM {
    mount_path_pseudo = true;
    NFS_Port = 2049;
    MNT_Port = 20048;
    NLM_Port = 32803;
    Rquota_Port = 875; 
    fsid_device = true;
}

LOG {
    Default_Log_Level = %s;
    Format {
        date_format = none;
        time_format = none;
        EPOCH = false;
        CLIENTIP = false; 
        HOSTNAME = false;
        PROGNAME = false;
        PID = false;
        THREAD_NAME = true;
        FILE_NAME = false;
        LINE_NUM = false;
        FUNCTION_NAME = false;
        COMPONENT = false;
        LEVEL = true; 
        OP_ID = false; 
        CLIENT_REQ_XID = false; 
        LOG_INDEX = false; 
    }
}

NFSV4 {
    Graceless = true;
    Allow_Numeric_Owners = true;
    Only_Numeric_Owners = true;
}
DIRECTORY_SERVICES {
    DomainName = "%s";
    Idmapping_Active = false;
}

EXPORT_DEFAULTS {
    Squash = All_Squash;
    Anonymous_uid = %d;
    Anonymous_gid = %d;
}

EXPORT {
    Path = "%s";
    Pseudo = "/";
    Export_Id = 1; 
    Access_Type = RO;
    Protocols = 3, 4;
    Transports = UDP, TCP; 
    SecType = sys;
	Squash = all_squash;
    Anonymous_uid = %d;
    Anonymous_gid = %d;
    FSAL {
        Name = VFS;
        fsid_type = uuid;
    }
}`

var (
	programPath = "/usr/bin/ganesha.nfsd"
	ganeshaConf = "/etc/ganesha/ganesha.conf"
	socketPath  = "/run/dbus/system_bus_socket"
)

type Struct struct {
	sState        systeminterface.SysStateEnum
	ganeshaCmd    *exec.Cmd
	logWriter     *helpers.SubsystemWriter
	dbusListener  net.Listener
	readyChan     chan struct{}
	zeroFreeSpace bool
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Config(cm config.ConfigMap) {
}

func (s *Struct) Setup() {
	logger.Debug(Name, "System Setup...")

	if err := s.writeGaneshaConfig(); err != nil {
		logger.Fatal(Name, "failed to execute master ganesha config file write utility", err)
	}
	logger.Debug(Name, "[write config]")

	if info, err := os.Stat(config.ShareRoot); err == nil {
		perms := info.Mode().Perm()

		if sysData, ok := info.Sys().(*syscall.Stat_t); ok {
			logger.Debug(Name, fmt.Sprintf(
				"CRUCIAL DIAGNOSTIC: Path [%s] has Perms [%04o] | Owner UID [%d] | Group GID [%d]",
				config.ShareRoot, perms, sysData.Uid, sysData.Gid,
			))
		}
	} else {
		logger.Error(Name, "Failed to read diagnostic stats for ShareRoot", err)
	}

	s.sState = systeminterface.SETUP
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) writeGaneshaConfig() error {

	detectedUID := 65534
	detectedGID := 65534

	if info, err := os.Stat(config.ShareRoot); err == nil {
		if sysData, ok := info.Sys().(*syscall.Stat_t); ok {
			detectedUID = int(sysData.Uid)
			detectedGID = int(sysData.Gid)
		}
	}

	gLogLevel := "EVENT"
	if logger.IsDebugActive(Name) {
		gLogLevel = "FULL_DEBUG"
	}

	configContent := fmt.Sprintf(CONFIGFILE, gLogLevel, config.DomainName, detectedUID, detectedGID, config.ShareRoot, detectedUID, detectedGID)

	return os.WriteFile(ganeshaConf, []byte(configContent), 0644)
}

func (s *Struct) Start() error {
	logger.Debug(Name, "System Starting...")
	if err := s.StartProgram(); err != nil {
		return err
	}
	if err := s.Wait(); err != nil {
		return err
	}
	s.sState = systeminterface.STARTED
	logger.Debug(Name, "[DONE]")
	return nil
}

func (s *Struct) Stop() {
	if s.ganeshaCmd != nil && s.ganeshaCmd.Process != nil {
		logger.Debug(Name, "Stopping NFS...")

		_ = s.ganeshaCmd.Process.Signal(syscall.SIGTERM)

		go func(p *os.Process) {
			time.Sleep(150 * time.Millisecond)
			if err := p.Signal(syscall.Signal(0)); err == nil {
				logger.Debug(Name, "NFS daemon socket locked. Enforcing asynchronous SIGKILL override pass.")
				_ = p.Kill()
			}
		}(s.ganeshaCmd.Process)

		_ = s.ganeshaCmd.Wait()
		logger.Debug(Name, "[CMD Stop]")
	}

	if s.logWriter != nil {
		_ = s.logWriter.Close()
		logger.Debug(Name, "[STDOUT->LOG STOP]")
	}

	s.sState = systeminterface.STOPPED
	logger.Debug(Name, "[DONE]")
}

func (s *Struct) Healthcheck() error {
	if s.ganeshaCmd == nil || s.ganeshaCmd.Process == nil {
		return fmt.Errorf("NFS SHARE is not initialized")
	}
	return s.ganeshaCmd.Process.Signal(syscall.Signal(0))
}
