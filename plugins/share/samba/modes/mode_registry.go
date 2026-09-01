package modes

import (
	"bytes"
	"fmt"
	"gorogs/logger"
	"gorogs/plugins/share/samba/structs"
	"gorogs/plugins/share/samba/vars"
	"io"
	"os/exec"
	"strings"
)

type ModeRegistry struct {
	ConfigMap structs.ConfigMap
	SharesMap structs.ShareMap
}

func (m *ModeRegistry) Setup() error {
	configMap := structs.NewConfigMap(structs.ModeRegistry)
	logger.Debug(Name, "[NEW CONFIG STRUCT]")
	configMap.SetSection(structs.GlobalName, directoryconf)
	logger.Debug(Name, "[ADD GLOBAL SECTION DIRS]")
	configMap.SectionMerge(structs.GlobalName, RegSetup)
	logger.Debug(Name, "[REGISTRY ONLY SETTING]")

	if err := configMap.ToFile(vars.MasterConfigFile); err != nil {
		return err
	}
	logger.Debug(Name, "[SAVE CONFIG FILE]")

	if err := sharedRegistrySystemSetup(); err != nil {
		return err
	}
	logger.Debug(Name, "[INITAL REGISTRY SETUP]")
	if vars.BatchInjection {
		return m.injectAllToRegistryBatch()
	}
	return m.injectAllToRegistryLoop()
}

func (m *ModeRegistry) NotifyCreate(shareName string, path string) error {
	m.SharesMap[shareName] = structs.NewShare(path)
	return m.SharesMap[shareName].RegistryShareAdd(shareName)
}

func (m *ModeRegistry) NotifyRemove(shareName string) error {
	if _, exists := m.SharesMap[shareName]; exists {
		delete(m.SharesMap, shareName)
		return m.SharesMap.RegistryShareDelete(shareName)
	}
	return fmt.Errorf("Share Not Found in List to Remove: %s", shareName)
}

func (m *ModeRegistry) NotifyCommentUpdate(shareName, comment string) error {
	if share, exists := m.SharesMap[shareName]; exists {
		share.Comment = comment
		m.SharesMap[shareName] = share
		return share.RegistryShareUpdate(shareName, "comment", comment)
	}
	return fmt.Errorf("Share Not Found in List to Update Comment: %s", shareName)
}

func (m *ModeRegistry) injectAllToRegistryBatch() error {
	cmdImport := exec.Command(vars.NetPath, "conf", "import", "/dev/stdin", "-s", vars.MasterConfigFile)
	logger.Debug(Name, "[BATCH:COMMAND]")
	globalReader := bytes.NewReader(m.ConfigMap.ToByte())
	logger.DebugF(Name, "[BATCH:GLOBAL COUNT %d]", m.ConfigMap.Count())
	sharesReader := bytes.NewReader(m.SharesMap.ToByte())
	logger.DebugF(Name, "[BATCH:SHARES COUNT %d]", m.SharesMap.Count())
	cmdImport.Stdin = io.MultiReader(globalReader, sharesReader)
	logger.Debug(Name, "[BATCH:MULTIREADER]")
	output, err := cmdImport.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Atomic global registry bulk import failed: %s ERROR: %w", strings.TrimSpace(string(output)), err)
	}
	logger.DebugF(Name, "[BATCH:INJECTED:%s]", output)
	return nil
}

func (m *ModeRegistry) injectAllToRegistryLoop() error {
	globals := m.ConfigMap.GetSection(structs.GlobalName)
	logger.Debug(Name, "[LOOP:GLOBALS]")
	for parameter, val := range globals {
		cmdGlobal := exec.Command(vars.NetPath, "conf", "setparm", "global", parameter, val, "-s", vars.MasterConfigFile)
		output, err := cmdGlobal.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to write Global parameter [%s:%s] ERROR: %w", parameter, strings.TrimSpace(string(output)), err)
		}
		logger.DebugF(Name, "[LOOP:ISTR:%s=%s:%s]", parameter, val, output)
	}

	logger.Debug(Name, "[LOOP:SHARES]")
	for entryName, entry := range m.SharesMap {
		if err := entry.RegistryShareAdd(entryName); err != nil {
			return fmt.Errorf("Failed to write Share [%s] ERROR: %w", entryName, err)
		}
		logger.DebugF(Name, "[LOOP:ADDED %s]", entryName)
	}
	logger.Debug(Name, "[LOOP:DONE]")
	return nil
}
