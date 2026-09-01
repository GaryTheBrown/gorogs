package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gorogs/config"
	"gorogs/logger"
)

func (s *Struct) StartProgram() error {
	s.readyChan = make(chan struct{})
	ganeshaArgs := []string{"-F", "-f", ganeshaConf, "-L", "/dev/stdout"}
	if logger.IsDebugActive(Name) {
		ganeshaArgs = append(ganeshaArgs, "-x", "-N", "FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command("/usr/bin/ganesha.nfsd", ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	if s.zeroFreeSpace {
		s.ganeshaCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libfake_statfs.so")
		logger.Debug(Name, "[ZERO FREE SPACE ENABLED]")
	}

	s.logWriter = config.NewSubsystemWriter(Name, nfsLogStripper)
	s.ganeshaCmd.Stdout = s.logWriter
	s.ganeshaCmd.Stderr = s.logWriter
	logger.Debug(Name, "[LINK STDOUT->LOG]")

	if err := s.ganeshaCmd.Start(); err != nil {
		close(s.readyChan)
		if s.logWriter != nil {
			_ = s.logWriter.Close()
		}
		return fmt.Errorf("failed to initialize background ganesha.nfsd: %w", err)
	}
	logger.Debug(Name, "[CMD START]")

	return nil
}

func nfsLogModeStringToLoggerType(strType string) config.LogType {
	switch strings.TrimSpace(strType) {
	case "LOG":
		return config.LOGINFO
	case "EVENT":
		return config.LOGINFO
	case "INFO":
		return config.LOGINFO
	case "WARN":
		return config.LOGWARN
	case "MAJ":
		return config.LOGERROR
	case "CRIT":
		return config.LOGERROR
	case "FATAL":
		return config.LOGFATAL
	case "DEBUG":
		return config.LOGDEBUG
	case "MID_DEBUG":
		return config.LOGDEBUG
	case "M_DBG":
		return config.LOGDEBUG
	case "FULL_DEBUG":
		return config.LOGDEBUG
	case "F_DBG":
		return config.LOGDEBUG
	case "NULL":
		return config.LOGNONE
	default:
		return config.LOGNONE
	}
}

var (
	stringShortMode bool = false
	stringFatalMode bool = false
)

func nfsLogStripper(line string) (string, config.LogType, string) {
	if line != "" {
		if stringFatalMode {
			return "", config.LOGERROR, line
		}
		if !stringShortMode {
			if line[0] == '[' {
				stringShortMode = true
			} else {
				_, after, _ := strings.Cut(line, "[")
				subsystem, after, _ := strings.Cut(after, "] ")

				splice := strings.SplitAfterN(after, ":", 3)
				if strings.ToLower(subsystem) == "main" {
					subsystem = ""
				}
				loggerType := nfsLogModeStringToLoggerType(splice[1])
				if loggerType == config.LOGFATAL {
					stringFatalMode = true
				}
				return subsystem, loggerType, splice[2]
			}
		}
		if stringShortMode {
			subsystem, afterBrackets, _ := strings.Cut(line[1:], "]")
			if strings.ToLower(subsystem) == "main" {
				subsystem = ""
			}
			msgType, msg, _ := strings.Cut(afterBrackets, ":")
			loggerType := nfsLogModeStringToLoggerType(msgType)
			if loggerType == config.LOGFATAL {
				stringFatalMode = true
			}
			return subsystem, nfsLogModeStringToLoggerType(msgType), msg
		}
	}
	return "", config.LOGNONE, "string"
}
