package samba

import (
	"syscall"
	"time"

	"gorogs/logger"
	"gorogs/plugins/share/samba/vars"
	"gorogs/system/helpers"
)

func (s *Struct) WaitForStart(maxWait time.Duration) bool {
	currentTick := 100 * time.Millisecond
	startTime := time.Now()
	probeAttempts := 0
	logger.Debug(Name, "[WAIT")
	for time.Since(startTime) < maxWait {
		probeAttempts++
		logger.Debug(Name, ".")
		if err := vars.Cmd.Process.Signal(syscall.Signal(0)); err != nil {
			if s.logWriter != nil {
				s.logWriter.Close()
			}
			logger.Fatal(Name, "samba smbd daemon process terminated unexpectedly during boot", err)
		}
		if helpers.WaitForSocket("tcp", "127.0.0.1:445", 50*time.Millisecond) {
			logger.Debug(Name, "DONE]")
			return true
		}
		if probeAttempts > 10 {
			currentTick = min(time.Duration(float64(currentTick)*1.5), 2*time.Second)
		}
		time.Sleep(currentTick)
	}
	return false
}
