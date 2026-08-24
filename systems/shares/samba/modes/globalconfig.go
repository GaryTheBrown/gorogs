package modes

import (
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
)

var (
	masterConf = structs.ConfigSection{
		"access based share enum": "yes",
		"aio read size":           "1",
		"csc policy":              "disable",
		"fake oplocks":            "yes",
		"guest only":              "yes",
		"level2 oplocks":          "yes",
		"load printers":           "no",
		"local master":            "no",
		"locking":                 "no",
		"log file":                "/dev/null",
		"log level":               "0",
		"map to guest":            "bad user",
		"map readonly":            "no",
		"multicast dns register":  "no",
		"name resolve order":      "host",
		"oplocks":                 "yes",
		"posix locking":           "no",
		"printcap name":           "/dev/null",
		"security":                "user",
		"server role":             "standalone",
		"server string":           "Read only Share",
		"smb encrypt":             "off",
		"strict locking":          "no",
		"usershare allow guests":  "yes",
		"use sendfile":            "yes",
		"veto files":              "/*.*/" + vars.VetoFiles,
		"workgroup":               config.Workgroup,
	}

	directoryconf = structs.ConfigSection{
		"lock directory":  "/var/lock/samba",
		"state directory": vars.SambaBaseLibDir,
		"cache directory": "/var/cache/samba",
		"private dir":     vars.InternalDBPath,
	}

	netbiosAdditions = structs.ConfigSection{
		"netbios name":     config.Hostname,
		"dns proxy":        "no",
		"hostname lookups": "no",
	}

	antiNetbiosAdditions = structs.ConfigSection{
		"disable netbios": "yes",
	}

	RegFailAdditions = structs.ConfigSection{
		"mdns name": "mdns",
	}

	RegSetup = structs.ConfigSection{
		"config backend": "registry",
	}
)

func SharedConfigFile(mode structs.ModeEnum) structs.ConfigMap {
	configMap := structs.NewConfigMap(mode)
	logger.DebugAppend(Name, "[NEW CONFIG STRUCT]")
	configMap.SetSection("global", masterConf)
	logger.DebugAppend(Name, "[SET GLOBAL SECTION]")
	if mode != structs.ModeRegistry {
		configMap.SectionMerge(structs.GlobalName, directoryconf)
		logger.DebugAppend(Name, "[MERGE DIR SETTINGS]")
	}
	if config.IsEnabled("netbios") {
		configMap.SectionMerge(structs.GlobalName, netbiosAdditions)
		logger.DebugAppend(Name, "[MERGE NETBIOS SETTINGS]")
	} else {
		if mode != structs.ModeRegistry {
			configMap.SectionMerge(structs.GlobalName, antiNetbiosAdditions)
			logger.DebugAppend(Name, "[MERGE ANTI NETBIOS SETTINGS]")
		}
	}

	if mode != structs.ModeRegistry {
		configMap.SectionMerge(structs.GlobalName, RegFailAdditions)
		logger.DebugAppend(Name, "[MERGE NOTINREG SETTINGS]")
	}
	logger.DebugAppend(Name, "[CONFIG STRUCT DONE]")
	return configMap

}
