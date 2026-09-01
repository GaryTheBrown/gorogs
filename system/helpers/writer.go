package helpers

import (
	"bytes"
	"fmt"
	"gorogs/logger"
	"sync"
)

type LogType uint8

const (
	LOGNONE LogType = iota
	LOGINFO
	LOGWARN
	LOGERROR
	LOGFATAL
	LOGDEBUG
)

type SubsystemWriter struct {
	mu         sync.Mutex
	loggerName string
	buffer     []byte

	stripFunc func(string) (string, LogType, string)
}

func NewSubsystemWriter(name string, stripFn func(string) (string, LogType, string)) *SubsystemWriter {
	return &SubsystemWriter{
		loggerName: name,
		stripFunc:  stripFn,
	}
}

func (sw *SubsystemWriter) Write(p []byte) (n int, err error) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	sw.buffer = append(sw.buffer, p...)

	for {
		idx := bytes.IndexByte(sw.buffer, '\n')
		if idx == -1 {
			break
		}

		line := string(sw.buffer[:idx])
		sw.buffer = sw.buffer[idx+1:]

		sw.processLine(line)
	}

	return len(p), nil
}

func (sw *SubsystemWriter) processLine(lineIn string) {
	if sw.stripFunc == nil {
		logger.Info(sw.loggerName, lineIn)
		return
	}
	fullName := sw.loggerName
	subsystem, logType, line := sw.stripFunc(lineIn)
	if subsystem != "" {
		fullName = fmt.Sprintf("%s.%s", sw.loggerName, subsystem)
	}
	switch logType {
	case LOGINFO:
		logger.Info(fullName, line)
	case LOGWARN:
		logger.Warn(fullName, line)
	case LOGERROR:
		logger.Error(fullName, line, fmt.Errorf("ERROR IN PROGRAM"))
	case LOGFATAL:
		logger.Fatal(fullName, line, fmt.Errorf("FATAL ISSUE IN PROGRAM"))
	case LOGDEBUG:
		logger.Debug(fullName, line)
	default:
		return
	}
}

func (sw *SubsystemWriter) Close() error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	if len(sw.buffer) > 0 {
		sw.processLine(string(sw.buffer))
		sw.buffer = nil
	}

	return nil
}
