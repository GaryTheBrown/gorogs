package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gorogs/beacons"
	"gorogs/beacons/wsdd"
	"gorogs/config"
	"gorogs/health"
	"gorogs/logger"
	"gorogs/shares"
	"gorogs/utils"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Launch listener and receive its structural lifecycle tracking channel
	listenerDone, err := incoming.StartUDPListener(ctx, "0.0.0.0:3702")
	if err != nil {
		log.Fatalf("Failed to initialize UDP engine: %v", err)
	}

	fmt.Println("File Server operating inside container environment...")

	// Listen for Docker stop (SIGTERM) or Terminal interrupts (SIGINT)
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// // Block here until Docker says stop
	// <-stopChan
	// fmt.Println("Docker stop signal detected! Safely draining connections...")

	// // Trigger the cleanup sequence
	// cancel()

	// // Block until the background threads confirm they have finished processing
	// <-listenerDone
	// fmt.Println("Container engine shut down gracefully. Exiting.")

	utils.InitializeRuntimeConfig()

	if config.IsCheckMode {
		health.RunHealthProbeClient()
		return
	}

	logger.Info("CORE", "Initializing master storage orchestration supervisor engine...")
	health.StartHealthServer()

	rpcbind := &beacons.RpcbindBeacon{}
	logger.Info("CORE", "Executing mandatory priority pre-flight checks for component: rpcbind")
	rpcErr := rpcbind.Setup()

	if rpcErr == beacons.ErrServiceDisabled {
		logger.Info("CORE", "RPCBIND setup notice: Service is deactivated via environment toggles.")
	} else if rpcErr != nil {
		logger.Fatal("CORE", "Critical initialization failure during priority rpcbind configuration setup phase", rpcErr)
	} else {
		// This blocks synchronously until port 111 is actively routing over the IPVLAN interface!
		if err := rpcbind.Start(); err != nil {
			logger.Fatal("CORE", "Failed to launch priority rpcbind daemon binary process tree", err)
		}
		health.TrackedBeacons["rpcbind"] = rpcbind
	}

	activeShares := []struct {
		name  string
		share shares.StorageShare
	}{
		{"nfs", &shares.NfsShare{}},
		{"samba", &shares.SambaShare{}},
	}

	for _, item := range activeShares {
		if item.name == "nfs" && !config.NfsEnabled {
			continue
		}
		if item.name == "samba" && !config.SambaEnabled {
			continue
		}

		logger.Info("CORE", "Executing setup checks for component: "+item.name)
		if err := item.share.Setup(); err != nil {
			logger.Fatal("CORE", "Critical initialization failure during share configuration setup phase", err)
		}

		// NFS will block here internally until the Ganesha logs confirm socket readiness
		if err := item.share.Start(); err != nil {
			logger.Fatal("CORE", "Failed to launch supervised daemon binary process tree", err)
		}

		health.TrackedShares[item.name] = item.share
	}

	activeBeacons := []struct {
		name   string
		beacon beacons.DiscoveryBeacon
	}{
		{"mdns", &beacons.MdnsBeacon{}},
		{"wsdd", &wsdd.WsddBeacon{}},
	}

	for _, item := range activeBeacons {
		if item.name == "wsdd" && !config.SambaEnabled {
			continue
		}

		logger.Info("CORE", "Executing setup checks for component: "+item.name)
		err := item.beacon.Setup()

		if err == beacons.ErrServiceDisabled {
			continue
		} else if err != nil {
			logger.Fatal("CORE", "Critical initialization failure during discovery beacon setup phase", err)
		}

		if err := item.beacon.Start(); err != nil {
			logger.Fatal("CORE", "Failed to arm advertisement handles or bind discovery sockets", err)
		}

		health.TrackedBeacons[item.name] = item.beacon
	}

	logger.Info("CORE", "All active operational layers successfully linked. Arming signal interception traps.")

	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan, syscall.SIGTERM, syscall.SIGINT)

	caughtSignal := <-shutdownSignalChan
	logger.Info("CORE", "Interception caught system event signal: "+caughtSignal.String()+". Commencing orderly cleanup procedures...")

	for name, beacon := range health.TrackedBeacons {
		logger.Info("CORE", "Dismantling network beacon handler channels: "+name)
		if err := beacon.Stop(); err != nil {
			logger.Error("CORE", "Teardown sequence returned errors during beacon resource release", err)
		}
	}

	for name, share := range health.TrackedShares {
		logger.Info("CORE", "Halting physical file system process tree: "+name)
		if err := share.Stop(); err != nil {
			logger.Error("CORE", "Teardown sequence returned errors during daemon execution release", err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	logger.Info("CORE", "All background worker threads reaped cleanly. Orchestrator runtime loop completed.")
}
