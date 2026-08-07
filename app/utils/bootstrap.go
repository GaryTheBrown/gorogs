package utils

import (
	"fmt"
	"net"
	"os"
	"strings"

	"gorogs/app/config"
	"gorogs/app/logger"
)

func InitializeRuntimeConfig() {
	logger.RegisterDebugTargets(os.Getenv("DEBUG_LOG"))

	// Explicitly target index 1 to resolve the string slice validation mismatch
	isCheck := len(os.Args) > 1 && os.Args[1] == "--check-health"

	_, hasNfsKill := os.LookupEnv("DISABLE_NFS")
	_, hasSambaKill := os.LookupEnv("DISABLE_SAMBA")
	_, hasRpcbindKill := os.LookupEnv("DISABLE_RPCBIND")
	_, hasWsddKill := os.LookupEnv("DISABLE_WSDD")
	_, hasMdnsKill := os.LookupEnv("DISABLE_ZEROCONF")
	_, hasLiveKill := os.LookupEnv("DISABLE_LIVECHANGES")

	hEnv := strings.ToLower(os.Getenv("HEALTH_MODE"))
	var resolvedLevel config.HealthLevel

	switch hEnv {
	case "full":
		resolvedLevel = config.LevelFull
	case "critical":
		resolvedLevel = config.LevelCritical
	case "shares":
		resolvedLevel = config.LevelShares
	case "nfs":
		resolvedLevel = config.LevelNfs
	case "samba":
		resolvedLevel = config.LevelSamba
	case "disabled":
		resolvedLevel = config.LevelDisabled
	default:
		resolvedLevel = config.LevelDefault
	}

	nodeName, err := os.Hostname()
	if err != nil || nodeName == "" {
		nodeName = os.Getenv("HOSTNAME")
		if nodeName == "" {
			nodeName = "storagenode"
		}
	}

	cIP, gIP, err := QueryNetworkLayout()
	if err != nil {
		logger.Fatal("CONFIG", "Foundational network table routing query failed. Halting setup initialization.", err)
	}

	detectedDomain := "local"
	if gIP != nil && !gIP.IsLoopback() {
		names, err := net.LookupAddr(gIP.String())
		if err == nil && len(names) > 0 {
			fqdn := strings.TrimSuffix(names[0], ".")
			parts := strings.Split(fqdn, ".")
			if len(parts) > 1 {
				detectedDomain = strings.Join(parts[1:], ".")
			}
		}
	}

	hostsContent := fmt.Sprintf("127.0.0.1 localhost\n::1 localhost\n%s %s %s.%s\n", cIP.String(), nodeName, nodeName, detectedDomain)
	if err := os.WriteFile("/etc/hosts", []byte(hostsContent), 0644); err != nil {
		logger.Error("CONFIG", "Warning: Failed to construct runtime network boundary resolution mapping inside /etc/hosts", err)
	}

	// Generating a clean, un-truncated nsswitch configuration to force local file resolution matches first
	nsswitchContent := `passwd:         files
group:          files
shadow:         files
gshadow:        files

hosts:          files dns

networks:       files
protocols:      db files
services:       db files
ethers:         db files
rpc:            db files
`
	if err := os.WriteFile("/etc/nsswitch.conf", []byte(nsswitchContent), 0644); err != nil {
		logger.Error("CONFIG", "Warning: Failed to construct mandatory operating system resolution priority layer inside /etc/nsswitch.conf", err)
	}

	config.Instance = &config.HubConfig{
		Name:               nodeName,
		DomainSuffix:       detectedDomain,
		ContainerIP:        cIP,
		HostGateway:        gIP,
		IsCheckMode:        isCheck,
		HealthMode:         resolvedLevel,
		NfsEnabled:         !hasNfsKill,
		SambaEnabled:       !hasSambaKill,
		RpcbindEnabled:     !hasRpcbindKill && !hasNfsKill,
		WsddEnabled:        !hasWsddKill && !hasSambaKill,
		MdnsEnabled:        !hasMdnsKill,
		LiveChangesEnabled: !hasLiveKill,
	}

	if isCheck {
		return
	}

	logger.Info("CONFIG", "Boot evaluation: Commencing structural system configuration analysis...")
	logger.Info("CONFIG", fmt.Sprintf("Health Check Strategy Configured: Mode Level [%d]", resolvedLevel))
	logger.Info("CONFIG", fmt.Sprintf("Evaluated service matrices: NFS=%v, SMB=%v, RPCBIND=%v, WSDD=%v, MDNS=%v",
		config.Instance.NfsEnabled, config.Instance.SambaEnabled, config.Instance.RpcbindEnabled, config.Instance.WsddEnabled, config.Instance.MdnsEnabled))
	logger.Info("CONFIG", fmt.Sprintf("Network bounds validated: Node Unicast IP=%s, Domain FQDN=%s.%s", cIP.String(), nodeName, detectedDomain))
}
