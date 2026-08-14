package main

import (
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

	utils.InitializeRuntimeConfig()

	if config.IsCheckMode {
		health.RunHealthProbeClient()
		return
	}

	beaconConfig := beacons.AppConfig{
		ServerName:   config.Name,
		DomainSuffix: config.DomainSuffix,
		ContainerIP:  config.ContainerIP,
		HostGateway:  config.HostGateway,
	}
	logger.Info("CORE", "Initializing master storage orchestration supervisor engine...")
	health.StartHealthServer()

	rpcbind := &beacons.RpcbindBeacon{}
	logger.Info("CORE", "Executing mandatory priority pre-flight checks for component: rpcbind")
	rpcErr := rpcbind.Setup(beaconConfig)

	if rpcErr == beacons.ErrServiceDisabled {
		logger.Info("CORE", "RPCBIND setup notice: Service is deactivated via environment toggles.")
	} else if rpcErr != nil {
		logger.Fatal("CORE", "Critical initialization failure during priority rpcbind configuration setup phase", rpcErr)
	} else {
		if err := rpcbind.Start(); err != nil {
			logger.Fatal("CORE", "Failed to launch priority rpcbind daemon binary process tree", err)
		}
		health.TrackedBeacons["rpcbind"] = rpcbind
	}

	// if config.NmbdEnabled {
	// 	nmbdBeacon := &beacons.NetBIOSBeacon{}
	// 	logger.Info("CORE", "Executing mandatory priority pre-flight checks for component: nmbd")

	// 	if err := nmbdBeacon.Setup(beaconConfig); err != nil {
	// 		logger.Fatal("CORE", "Critical initialization failure during priority nmbd configuration setup phase", err)
	// 	}

	// 	if err := nmbdBeacon.Start(); err != nil {
	// 		logger.Fatal("CORE", "Failed to launch priority nmbd beacon binary process tree", err)
	// 	}
	// 	health.TrackedBeacons["nmbd"] = nmbdBeacon
	// } else {
	// 	logger.Info("CORE", "NMBD setup notice: NetBIOS discovery beacon is deactivated via environment toggles.")
	// }

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
		err := item.beacon.Setup(beaconConfig)

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
