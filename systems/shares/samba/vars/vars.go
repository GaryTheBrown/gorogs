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
)

var (
	BaseDirOverlay      bool
	BatchInjection      bool
	VetoFiles           string
	DefaultShareComment string
	ServerComment       string
)

var (
	Cmd *exec.Cmd
)
