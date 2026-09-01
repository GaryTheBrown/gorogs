package main

import (
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/system"
	"os"
	"os/signal"
	"syscall"
)

const (
	logName string = "gorogs"
)

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

	logger.Info(logName, "Discovering and Loading Systems Plugins...")
	if err := system.LoadPlugins("/usr/lib/gorogs"); err != nil {
		logger.Fatal(logName, "FATAL: System plugin engine failed to bootstrap", err)
	}

	logger.Info(logName, "Configs Setup...")
	system.Config()
	logger.Info(logName, "Setting up Systems...")
	system.Setup()
	logger.Info(logName, "Starting Systems...")
	system.Start()
	logger.Info(logName, "Systems Started")

	logger.Info(logName, "Adding Systems To Healthcheck...")
	for _, sys := range system.SystemList {
		healthcheck.AddTracker(sys)
	}

	logger.Info(logName, "Arming signal interception traps.")
	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan, syscall.SIGTERM, syscall.SIGINT)

	caughtSignal := <-shutdownSignalChan
	logger.InfoF(logName, "Interception caught system event signal: %s Commencing orderly cleanup procedures...", caughtSignal.String())

	logger.Info(logName, "Stopping Systems...")
	system.Stop()
	logger.Info(logName, "Systems Stopped")

	logger.Info(logName, "Stopping Healthcheck...")
	healthcheck.Stop()
	logger.Info(logName, "Healthcheck Stopped")

	logger.Info(logName, "All background worker threads reaped cleanly. Shutdown Clean.")
	logger.Close()

	syscall.Exit(0)
}
