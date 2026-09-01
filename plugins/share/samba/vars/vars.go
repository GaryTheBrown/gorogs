package vars

import "os/exec"

const (
	ProgramPath      string = "/usr/bin/smbd"
	NetPath          string = "/usr/bin/net"
	SmbpasswdPath    string = "/usr/bin/smbpasswd"
	DBWrapToolPath   string = "/usr/bin/dbwrap_tool"
	SambaBaseLibDir  string = "/var/lib/samba"
	InternalDBPath   string = SambaBaseLibDir + "/private"
	InternalDBFile   string = SambaBaseLibDir + "/registry.tdb"
	RegistryDBFile   string = SambaBaseLibDir + "/smbconf.tdb"
	MasterConfigFile string = "/etc/samba/smb.conf"
	ShareConfigFile  string = SambaBaseLibDir + "/smb-shares.conf"
)

var (
	BaseDirOverlay      bool
	BatchInjection      bool
	ZeroFreeSpace       bool
	VetoFiles           string
	DefaultShareComment string
	ServerComment       string

	Cmd *exec.Cmd
)
