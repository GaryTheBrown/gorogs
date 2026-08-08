package beacons

import (
	"fmt"

	"gorogs/config"
	"gorogs/logger"

	"github.com/grandcat/zeroconf"
)

type MdnsBeacon struct {
	servers []*zeroconf.Server
}

func (m *MdnsBeacon) Setup() error {
	logger.Info("MDNS", "Evaluating network discovery broadcast requirements...")

	if !config.Instance.MdnsEnabled {
		logger.Info("MDNS", "Global Zeroconf kill switch active. Bypassing mDNS beacon manager.")
		return ErrServiceDisabled
	}

	if !config.Instance.NfsEnabled && !config.Instance.SambaEnabled {
		logger.Info("MDNS", "No file storage shares are currently enabled. Bypassing unneeded mDNS server.")
		return ErrServiceDisabled
	}

	logger.Info("MDNS", "Pre-flight checks passed. Service registration profile is valid.")
	return nil
}

func (m *MdnsBeacon) Start() error {
	logger.Info("MDNS", "Initializing unified mDNS networking engine...")

	nodeName := config.Instance.Name
	containerIPStr := config.Instance.ContainerIP.String()

	// --- DYNAMIC FQDN DOMAIN SELECTION ---
	// Default back to standard multicast specs if bootstrap domain parsing fails
	domainTarget := "local."
	if config.Instance.DomainSuffix != "" && config.Instance.DomainSuffix != "local" {
		// Ensure it terminates with a trailing dot matching DNS protocol expectations
		domainTarget = config.Instance.DomainSuffix + "."
		logger.Info("MDNS", fmt.Sprintf("Using network router resolved FQDN suffix: %s", domainTarget))
	} else {
		logger.Info("MDNS", "Router provided no valid domain suffix. Falling back to default: local.")
	}
	// -------------------------------------

	if config.Instance.NfsEnabled {
		logger.Info("MDNS", fmt.Sprintf("Compiling unified advertisement record: [%s] -> NFS Protocol over %s:2049", nodeName, containerIPStr))

		logger.Debug("MDNS", "Invoking grandcat/zeroconf core API wrapper hooks for _nfs._tcp allocation")
		nfsSrv, err := zeroconf.Register(
			nodeName,
			"_nfs._tcp",
			domainTarget, // <-- FIXED: Passes the router resolved custom domain dynamically
			2049,
			[]string{},
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to register NFS service record profile: %w", err)
		}
		m.servers = append(m.servers, nfsSrv)
	}

	if config.Instance.SambaEnabled {
		logger.Info("MDNS", fmt.Sprintf("Compiling unified advertisement record: [%s] -> SMB Protocol over %s:445", nodeName, containerIPStr))

		logger.Debug("MDNS", "Invoking grandcat/zeroconf core API wrapper hooks for _smb._tcp allocation")
		smbSrv, err := zeroconf.Register(
			nodeName,
			"_smb._tcp",
			domainTarget, // <-- FIXED: Passes the router resolved custom domain dynamically
			445,
			[]string{},
			nil,
		)
		if err != nil {
			m.Stop()
			return fmt.Errorf("failed to register Samba service record profile: %w", err)
		}
		m.servers = append(m.servers, smbSrv)
	}

	logger.Info("MDNS", "Universal mDNS discovery beacons active and broadcasting over the physical wire.")
	return nil
}

func (m *MdnsBeacon) Healthcheck() error {
	if len(m.servers) == 0 {
		return fmt.Errorf("unified mDNS discovery beacon array is uninitialized or empty")
	}
	return nil
}

func (m *MdnsBeacon) IsCritical() bool {
	return false
}

func (m *MdnsBeacon) Stop() error {
	logger.Info("MDNS", "Initiating shutdown sequence on unified mDNS discovery channels...")

	for i, srv := range m.servers {
		if srv != nil {
			logger.Debug("MDNS", fmt.Sprintf("Shutting down active mDNS socket binding index reference: %d", i))
			srv.Shutdown()
		}
	}

	m.servers = nil
	logger.Info("MDNS", "mDNS broadcast beacons dropped cleanly from network space.")
	return nil
}
