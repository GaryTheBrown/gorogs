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

	stripFunc func(string) (LogType, string)
}

func NewSubsystemWriter(name string, stripFn func(string) (LogType, string)) *SubsystemWriter {
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

	logType, line := sw.stripFunc(lineIn)
	switch logType {
	case LOGINFO:
		logger.Info(sw.loggerName, line)
	case LOGWARN:
		logger.Warn(sw.loggerName, line)
	case LOGERROR:
		logger.Error(sw.loggerName, line, fmt.Errorf("ERROR IN PROGRAM"))
	case LOGFATAL:
		logger.Fatal(sw.loggerName, line, fmt.Errorf("FATAL ISSUE IN PROGRAM"))
	case LOGDEBUG:
		logger.Debug(sw.loggerName, line)
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
