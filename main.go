package main

import (
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/systems"
	"os"
	"os/signal"
	"syscall"
)

const logName = "main"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--check-health" {
		healthcheck.ProbeClient()
		return
	}
	logger.Info(logName, "Setting up Healthcheck...")
	healthcheck.Setup()
	logger.Info(logName, "Starting Healthcheck...")
	healthcheck.Start()
	logger.Info(logName, "Healthcheck Started")

	logger.Info(logName, "Setting up Systems...")
	systems.Setup()
	logger.Info(logName, "Starting Systems...")
	systems.Start()
	logger.Info(logName, "Systems Started")

	logger.Info(logName, "Arming signal interception traps.")
	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan, syscall.SIGTERM, syscall.SIGINT)

	caughtSignal := <-shutdownSignalChan
	logger.InfoF(logName, "Interception caught system event signal: %s Commencing orderly cleanup procedures...", caughtSignal.String())

	logger.Info(logName, "Stopping Systems...")
	systems.Stop()
	logger.Info(logName, "Systems Stopped")

	logger.Info(logName, "Stopping Healthcheck...")
	healthcheck.Stop()
	logger.Info(logName, "Healthcheck Stopped")

	logger.Info(logName, "All background worker threads reaped cleanly. Shutdown Clean.")
	logger.Close()

	syscall.Exit(0)
}
