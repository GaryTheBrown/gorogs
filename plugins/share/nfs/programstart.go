package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"gorogs/logger"
	"gorogs/subsystemwriter"
)

func (s *Struct) StartProgram() error {
	s.readyChan = make(chan struct{})
	ganeshaArgs := []string{"-F", "-f", ganeshaConf, "-L", "/dev/stdout"}
	if logger.IsDebugActive(Name) {
		ganeshaArgs = append(ganeshaArgs, "-x", "-N", "FULL_DEBUG")
	}

	s.ganeshaCmd = exec.Command(programPath, ganeshaArgs...)
	s.ganeshaCmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	logger.Debug(Name, "[CMD SETUP]")

	if s.zeroFreeSpace {
		s.ganeshaCmd.Env = append(os.Environ(), "LD_PRELOAD=/usr/lib/libfake_statfs.so")
		logger.Debug(Name, "[ZERO FREE SPACE ENABLED]")
	}

	s.logWriter = subsystemwriter.NewSubsystemWriter(Name, nfsLogStripper)
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

func nfsLogModeStringToLoggerType(strType string) subsystemwriter.LogType {
	switch strings.TrimSpace(strType) {
	case "LOG":
		return subsystemwriter.LOGINFO
	case "EVENT":
		return subsystemwriter.LOGINFO
	case "INFO":
		return subsystemwriter.LOGINFO
	case "WARN":
		return subsystemwriter.LOGWARN
	case "MAJ":
		return subsystemwriter.LOGERROR
	case "CRIT":
		return subsystemwriter.LOGERROR
	case "FATAL":
		return subsystemwriter.LOGFATAL
	case "DEBUG":
		return subsystemwriter.LOGDEBUG
	case "MID_DEBUG":
		return subsystemwriter.LOGDEBUG
	case "M_DBG":
		return subsystemwriter.LOGDEBUG
	case "FULL_DEBUG":
		return subsystemwriter.LOGDEBUG
	case "F_DBG":
		return subsystemwriter.LOGDEBUG
	case "NULL":
		return subsystemwriter.LOGNONE
	default:
		return subsystemwriter.LOGNONE
	}
}

var (
	stringShortMode bool = false
	stringFatalMode bool = false
)

func nfsLogStripper(line string) (string, subsystemwriter.LogType, string) {
	if line != "" {
		if stringFatalMode {
			return "", subsystemwriter.LOGERROR, line
		}
		if !stringShortMode {
			if line[0] == '[' {
				stringShortMode = true
			} else {
				_, after, _ := strings.Cut(line, "[")
				subsystem, after, _ := strings.Cut(after, "] ")
				if strings.ToLower(subsystem) == "main" {
					subsystem = ""
				}

				splice := strings.SplitAfterN(after, ":", 3)
				if len(splice) == 1 {
					return subsystem, subsystemwriter.LOGWARN, splice[0]
				}
				loggerType := nfsLogModeStringToLoggerType(splice[1])
				if loggerType == subsystemwriter.LOGFATAL {
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
			if loggerType == subsystemwriter.LOGFATAL {
				stringFatalMode = true
			}
			return subsystem, nfsLogModeStringToLoggerType(msgType), msg
		}
	}
	return "", subsystemwriter.LOGNONE, "string"
}
