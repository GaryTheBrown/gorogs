package main

import (
	"gorogs/healthcheck"
	"gorogs/logger"
	"gorogs/system"
	"gorogs/system/systeminterface"
	"os"
	"os/signal"
	"syscall"

	_ "gorogs/plugins/beacon/netbios"
	_ "gorogs/plugins/beacon/rpcbind"
	_ "gorogs/plugins/beacon/wsdiscovery"
	_ "gorogs/plugins/beacon/zeroconf"
	_ "gorogs/plugins/share/nfs"
	_ "gorogs/plugins/share/samba"
)

const logName = "gorogs"

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

	logger.Info(logName, "Getting Systems ...")
	systeminterface.InitializeAndSort()
	logger.Info(logName, "Configs Setup...")
	system.Config()
	logger.Info(logName, "Setting up Systems...")
	system.Setup()
	logger.Info(logName, "Starting Systems...")
	system.Start()
	logger.Info(logName, "Systems Started")

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
