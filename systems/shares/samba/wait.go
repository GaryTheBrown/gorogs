package samba

import (
	"syscall"
	"time"

	"gorogs/logger"
	"gorogs/systems/helpers"
	"gorogs/systems/shares/samba/vars"
)

func (s *Struct) WaitForStart(maxWait time.Duration) bool {
	currentTick := 100 * time.Millisecond
	startTime := time.Now()
	probeAttempts := 0
	logger.DebugAppend(Name, "[WAIT")
	for time.Since(startTime) < maxWait {
		probeAttempts++
		logger.DebugAppend(Name, ".")
		if err := vars.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
			if s.logWriter != nil {
				s.logWriter.Close()
			}
			logger.Fatal(Name, "samba smbd daemon process terminated unexpectedly during boot", err)
		}
		if helpers.WaitForSocket("tcp", "127.0.0.1:445", 50*time.Millisecond) {
			logger.DebugAppend(Name, "DONE]")
			return true
		}
		if probeAttempts > 10 {
			currentTick = min(time.Duration(float64(currentTick)*1.5), 2*time.Second)
		}
		time.Sleep(currentTick)
	}
	return false
}
