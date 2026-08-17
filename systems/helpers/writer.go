package helpers

import (
	"bytes"
	"gorogs/logger"
	"strings"
	"sync"
)

type SubsystemWriter struct {
	mu         sync.Mutex
	loggerName string
	buffer     []byte

	matchPhrases []string
	readyChan    chan struct{}
	hasSignaled  bool

	stripFunc func(string) (string, bool)
}

func NewSubsystemWriter(name string, readyChan chan struct{}, matchPhrases []string, stripFn func(string) (string, bool)) *SubsystemWriter {
	return &SubsystemWriter{
		loggerName:   name,
		readyChan:    readyChan,
		matchPhrases: matchPhrases,
		stripFunc:    stripFn,
	}
}

func (w *SubsystemWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.buffer = append(w.buffer, p...)

	for {
		idx := bytes.IndexByte(w.buffer, '\n')
		if idx == -1 {
			break
		}

		line := string(w.buffer[:idx])
		w.buffer = w.buffer[idx+1:]

		w.processLine(line)
	}

	return len(p), nil
}

func (w *SubsystemWriter) processLine(line string) {
	if !w.hasSignaled && w.readyChan != nil {
		for _, phrase := range w.matchPhrases {
			if strings.Contains(line, phrase) {
				close(w.readyChan)
				w.hasSignaled = true
				break
			}
		}
	}

	if !logger.IsDebugActive(w.loggerName) {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			logger.Info(w.loggerName, trimmed)
		}
		return
	}

	if w.stripFunc != nil {
		var keep bool
		line, keep = w.stripFunc(line)
		if !keep {
			return
		}
	}

	trimmed := strings.TrimSpace(line)
	if trimmed != "" {
		logger.Debug(w.loggerName, trimmed)
	}
}

func (w *SubsystemWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) > 0 {
		w.processLine(string(w.buffer))
		w.buffer = nil
	}

	if !w.hasSignaled && w.readyChan != nil {
		close(w.readyChan)
		w.hasSignaled = true
	}
	return nil
}
