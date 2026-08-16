package main

import (
	"gorogs/config"
	"gorogs/health"
	"gorogs/logger"
	"gorogs/systems"
	"gorogs/systems/beacons/netbios"
	"gorogs/systems/beacons/rpcbind"
	"gorogs/systems/beacons/wsdiscovery"
	"gorogs/systems/beacons/zeroconf"
	"gorogs/systems/shares/nfs"
	"gorogs/systems/shares/samba"
	"gorogs/systems/utilities/zerospace"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type SystemNameEnum uint16

const (
	ZeroSpace SystemNameEnum = iota
	RpcBind
	NFS
	NetBIOS
	Samba
	WSDiscovery
	ZeroCONF
)

var (
	systemList = []systems.System{
		//ORDER IS IMPORTAND DON'T CHANGE THIS ORDER
		&zerospace.ZeroSpaceStruct{},
		&rpcbind.RPCBindStruct{},
		&nfs.NFSStruct{},
		&netbios.NetBIOSStruct{},
		&samba.SambaStruct{},
		&wsdiscovery.WSDiscoveryStruct{},
		&zeroconf.ZeroconfStruct{},
	}
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--check-health" {
		health.ProbeClient()
		return
	}

	healthChecker := health.CheckStruct{}
	healthChecker.Setup()
	healthChecker.Start()

	for i, sys := range systemList {
		sysAutoStart := sys.AutoStart()
		if sysAutoStart {
			sysDisabled := config.IsDisabled(sys.Name())
			if i == int(RpcBind) {
				if sysDisabled {
					if (systemList[NFS].AutoStart() && config.IsDisabled("nfs")) ||
						(!systemList[NFS].AutoStart() && !config.IsEnabled("nfs")) {
						continue
					}
				}
			} else if sysDisabled {
				continue
			}
		} else {
			sysEnabled := config.IsEnabled(sys.Name())
			if i == int(RpcBind) {
				if (systemList[NFS].AutoStart() && config.IsDisabled("nfs")) ||
					(!systemList[NFS].AutoStart() && !config.IsEnabled("nfs")) {
					continue
				}
			} else if !sysEnabled {
				continue
			}
		}
		if err := sys.Setup(); err != nil {
			//ERROR DO WE STOP. OF COURSE WE ARE
		}
	}

	for _, sys := range systemList {
		if sys.State(systems.SETUP) {
			if err := sys.Start(); err != nil {
				//Problem Starting system
			}
			if !healthChecker.AddTracker(sys) {
				//problem with adding to tracker
			}
		}
	}

	logger.Info("CORE", "All active operational layers successfully linked. Arming signal interception traps.")

	shutdownSignalChan := make(chan os.Signal, 1)
	signal.Notify(shutdownSignalChan, syscall.SIGTERM, syscall.SIGINT)

	caughtSignal := <-shutdownSignalChan
	logger.InfoF("CORE", "Interception caught system event signal: %s Commencing orderly cleanup procedures...", caughtSignal.String())

	for i := len(systemList) - 1; i >= 0; i-- {
		sys := systemList[i]
		if sys.State(systems.STARTED) {
			logger.InfoContinueF(sys.Name(), "Stopping [%s] system...", sys.Name())
			sys.Stop()
			logger.InfoEnd(sys.Name(), "[DONE]")
		}

	}
	time.Sleep(500 * time.Millisecond)
	logger.Info("CORE", "All background worker threads reaped cleanly. Orchestrator runtime loop completed.")
}
