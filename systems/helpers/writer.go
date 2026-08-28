package helpers

import (
	"bytes"
	"sync"
)

type SubsystemWriter struct {
	mu         sync.Mutex
	loggerName string
	buffer     []byte

	stripFunc func(string) (string, bool)
}

func NewSubsystemWriter(name string, stripFn func(string) (string, bool)) *SubsystemWriter {
	return &SubsystemWriter{
		loggerName: name,
		stripFunc:  stripFn,
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
	if w.stripFunc != nil {
		var keep bool
		line, keep = w.stripFunc(line)
		if !keep {
			return
		}
	}
}

func (w *SubsystemWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) > 0 {
		w.processLine(string(w.buffer))
		w.buffer = nil
	}

	return nil
}
