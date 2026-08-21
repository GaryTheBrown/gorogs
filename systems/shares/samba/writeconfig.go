package samba

import (
	"fmt"
	"os"
)

func (s *Struct) writeMasterSambaConfig() error {
	masterContent := fmt.Sprintf(`[global]
    config backend = registry
    lock directory = /var/lock/samba
    state directory = %s
    cache directory = /var/cache/samba
    private dir = %s
    log file = /dev/null
    max log size = 0
    log level = 0
    guest account = smbguest
    map to guest = bad user

[IPC$]
    path = /var/run/samba
    comment = IPC Service
    guest ok = yes
    read only = yes
    browseable = yes
`, sambaBaseLibDir, internalDBPath)

	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
}

// func (s *Struct) writeMasterSambaConfig() error {
// 	masterContent := fmt.Sprintf(`[global]
//     # Core Server Identity Block
//     workgroup = %s
//     server string = Read only Share
//     netbios name = %s
//     security = user
//     map to guest = bad user
//     usershare allow guests = yes
//     load printers = no
//     printcap name = /dev/null
//     logging = file
//     veto files = /.*/

//     # Direct the engine to read dynamic movie shares out of registry memory [index]
//     include = registry

//     # Explicitly lock folder environments inside your RAM disk mount space
//     lock directory = /var/lock/samba
//     state directory = %s
//     cache directory = /var/cache/samba
//     private dir = %s
// `, config.Workgroup, config.Hostname, sambaBaseLibDir, internalDBPath)

// 	return os.WriteFile(masterConfigPath, []byte(masterContent), 0644)
// }

// map to guest = bad user
// restrict anonymous = 0

// masterContent := "[global]\n" +
// 	"    netbios name = " + serverName + "\n" +
// 	"    server string = Read only Share\n" +
// 	"    log file = /var/log/samba/log.%%m\n" +
// 	"    max log size = 1000\n" +
// 	"    logging = file\n" +
// 	"    server role = standalone server\n" +
// 	"    map to guest = bad user\n" +
// 	"    usershare allow guests = yes\n" +
// 	"    usershare max shares = 0\n" +
// 	"    dns proxy = no\n" +
// 	"    hostname lookups = no\n" +
// 	"\n" +
// 	"    include = " + shareConfigPath + "\n"
