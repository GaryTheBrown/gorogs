package vars

import "os/exec"

const (
	ProgramPath     = "/usr/bin/smbd"
	NetPath         = "/usr/bin/net"
	SmbpasswdPath   = "/usr/bin/smbpasswd"
	DBWrapToolPath  = "/usr/bin/dbwrap_tool"
	SambaBaseLibDir = "/var/lib/samba"
	InternalDBPath  = SambaBaseLibDir + "/private"
	InternalDBFile  = SambaBaseLibDir + "/registry.tdb"
	RegistryDBFile  = SambaBaseLibDir + "/smbconf.tdb"

	MasterConfigFile = "/etc/samba/smb.conf"
	ShareConfigFile  = SambaBaseLibDir + "/smb-shares.conf"

	// DEFAULT CONFIG VARS TO EVENTUALLY BE LOADED IN FORM A map[string]any
	LibBaseDirOverlay   = true
	BatchInjection      = true
	VetoFiles           = "/*.~tmp/" //This one is for when copying big files to keep them hidden until complete.
	DefaultShareComment = ""
)

var (
	Cmd *exec.Cmd
)
