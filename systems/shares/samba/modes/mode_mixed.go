package modes

import (
	"bytes"
	"fmt"
	"gorogs/logger"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
	"os/exec"
)

type ModeMixed struct {
	ConfigMap structs.ConfigMap
	SharesMap structs.ShareMap
}

func (m *ModeMixed) Setup() error {
	if err := m.ConfigMap.ToFile(vars.MasterConfigFile); err != nil {
		return nil
	}
	logger.Debug(Name, "[WRITTEN CONFIG]")
	if err := sharedRegistrySystemSetup(); err != nil {
		return err
	}
	logger.Debug(Name, "[INITAL REGISTRY SETUP]")
	if vars.BatchInjection {
		return m.injectAllSharesToRegistryBatch()
	}
	return m.injectAllSharesToRegistryLoop()
}

func (m *ModeMixed) NotifyCreate(shareName string, path string) error {
	m.SharesMap[shareName] = structs.NewShare(path)
	return m.SharesMap[shareName].RegistryShareAdd(shareName)
}

func (m *ModeMixed) NotifyRemove(shareName string) error {
	if _, exists := m.SharesMap[shareName]; exists {
		delete(m.SharesMap, shareName)
		return m.SharesMap.RegistryShareDelete(shareName)
	}
	return fmt.Errorf("Share Not Found in List to Remove: %s", shareName)
}

func (m *ModeMixed) NotifyCommentUpdate(shareName, comment string) error {
	if share, exists := m.SharesMap[shareName]; exists {
		share.Comment = comment
		m.SharesMap[shareName] = share
		return share.RegistryShareUpdate(shareName, "comment", comment)
	}
	return fmt.Errorf("Share Not Found in List to Update Comment: %s", shareName)
}

func (m *ModeMixed) injectAllSharesToRegistryBatch() error {
	logger.Debug(Name, "[BATCH SHARE ADD]")
	cmdImport := exec.Command(vars.NetPath, "conf", "import", "/dev/stdin", "-s", vars.MasterConfigFile)
	cmdImport.Stdin = bytes.NewReader(m.SharesMap.ToByte())
	if output, err := cmdImport.CombinedOutput(); err != nil {
		return fmt.Errorf("IMPORT FAILED OUTPUT FROM COMMAND: '%s' ERROR: %s", output, err.Error())
	}
	return nil
}

func (m *ModeMixed) injectAllSharesToRegistryLoop() error {
	logger.Debug(Name, "[LOOP SHARE ADD][")
	for entryName, entry := range m.SharesMap {
		logger.Debug(Name, ".")
		if err := entry.RegistryShareAdd(entryName); err != nil {
			return fmt.Errorf("IMPORT FAILED OUTPUT FROM COMMAND:[%s] ERROR: %s", entryName, err.Error())
		}
	}
	logger.Debug(Name, "][DONE]")
	return nil
}
