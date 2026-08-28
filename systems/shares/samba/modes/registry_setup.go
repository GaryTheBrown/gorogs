package modes

import (
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func sharedRegistrySystemSetup() error {
	_ = os.WriteFile(filepath.Join(vars.SambaBaseLibDir, "account_policy.tdb"), []byte{}, 0600)
	logger.Debug(Name, "[CREATE ACCOUNT_POLICY.TDB]")
	_ = os.WriteFile(filepath.Join(vars.SambaBaseLibDir, "winbindd_idmap.tdb"), []byte{}, 0600)
	logger.Debug(Name, "[CREATE WINBINDD_IDMAP.TDB]")
	cmdPasswd := exec.Command(vars.SmbpasswdPath, "-L", "-c", vars.MasterConfigFile, "-a", "nobody", "-n")
	if output, err := cmdPasswd.CombinedOutput(); err != nil {
		return fmt.Errorf("User database initialization failed: %s ERROR: %w", strings.TrimSpace(string(output)), err)
	}
	logger.Debug(Name, "[CREATE USER.TDB]")

	cmdSID := exec.Command(vars.NetPath, "setlocalsid", "S-1-5-21-1111111111-2222222222-3333333333", "-s", vars.MasterConfigFile)
	if output, err := cmdSID.CombinedOutput(); err != nil {
		return fmt.Errorf("Failed to register machine identity tokens: %s ERROR: %w", strings.TrimSpace(string(output)), err)
	}
	logger.Debug(Name, "[ADD MACHINE ID]")
	return nil
}

func injectRegistryOverrides() error {

	confToAdd := structs.ConfigSection{}
	if !config.IsEnabled("netbios") {
		confToAdd.SectionMerge(antiNetbiosAdditions)
	}
	confToAdd.SectionMerge(RegFailAdditions)

	for param, value := range confToAdd {
		key := fmt.Sprintf("global:%s", param)

		cmd := exec.Command(vars.DBWrapToolPath, "--persistent", vars.RegistryDBFile, "store", key, "string", value)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to force-inject %s to TDB: %s ERROR: %w", param, strings.TrimSpace(string(output)), err)
		}
	}

	return nil
}
