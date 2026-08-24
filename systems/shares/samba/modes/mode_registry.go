package modes

import (
	"bytes"
	"fmt"
	"gorogs/logger"
	"gorogs/systems/shares/samba/structs"
	"gorogs/systems/shares/samba/vars"
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
	logger.DebugAppend(Name, "[NEW CONFIG STRUCT]")
	configMap.SetSection(structs.GlobalName, directoryconf)
	logger.DebugAppend(Name, "[ADD GLOBAL SECTION DIRS]")
	configMap.SectionMerge(structs.GlobalName, RegSetup)
	logger.DebugAppend(Name, "[REGISTRY ONLY SETTING]")

	if err := configMap.ToFile(vars.MasterConfigFile); err != nil {
		return err
	}
	logger.DebugAppend(Name, "[SAVE CONFIG FILE]")

	if err := sharedRegistrySystemSetup(); err != nil {
		return err
	}
	logger.DebugAppend(Name, "[INITAL REGISTRY SETUP]")
	if vars.BatchInjection {
		return m.injectAllToRegistryBatch()
	}
	return m.injectAllToRegistryLoop()
}

func (m *ModeRegistry) NotifyCreate(shareName string, path string) error {
	m.SharesMap[shareName] = structs.NewShare(path, vars.DefaultShareComment)
	return m.SharesMap[shareName].RegistryShareAdd(shareName)
}

func (m *ModeRegistry) NotifyRemove(shareName string) error {
	if _, exists := m.SharesMap[shareName]; exists {
		delete(m.SharesMap, shareName)
		return m.SharesMap.RegistryShareDelete(shareName)
	}
	return fmt.Errorf("Share Not Found in List to Remove: %s", shareName)
}

func (m *ModeRegistry) injectAllToRegistryBatch() error {
	cmdImport := exec.Command(vars.NetPath, "conf", "import", "/dev/stdin", "-s", vars.MasterConfigFile)
	logger.DebugAppend(Name, "[BATCH:COMMAND]")
	globalReader := bytes.NewReader(m.ConfigMap.ToByte())
	logger.DebugAppendF(Name, "[BATCH:GLOBAL COUNT %d]", m.ConfigMap.Count())
	sharesReader := bytes.NewReader(m.SharesMap.ToByte())
	logger.DebugAppendF(Name, "[BATCH:SHARES COUNT %d]", m.SharesMap.Count())
	cmdImport.Stdin = io.MultiReader(globalReader, sharesReader)
	logger.DebugAppend(Name, "[BATCH:MULTIREADER]")
	output, err := cmdImport.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Atomic global registry bulk import failed: %s ERROR: %w", strings.TrimSpace(string(output)), err)
	}
	logger.DebugAppendF(Name, "[BATCH:INJECTED:%s]", output)
	return nil
}

func (m *ModeRegistry) injectAllToRegistryLoop() error {
	globals := m.ConfigMap.GetSection(structs.GlobalName)
	logger.DebugAppend(Name, "[LOOP:GLOBALS]")
	for parameter, val := range globals {
		cmdGlobal := exec.Command(vars.NetPath, "conf", "setparm", "global", parameter, val, "-s", vars.MasterConfigFile)
		output, err := cmdGlobal.CombinedOutput()
		if err != nil {
			return fmt.Errorf("Failed to write Global parameter [%s:%s] ERROR: %w", parameter, strings.TrimSpace(string(output)), err)
		}
		logger.DebugAppendF(Name, "[LOOP:ISTR:%s=%s:%s]", parameter, val, output)
	}

	logger.DebugAppend(Name, "[LOOP:SHARES]")
	for entryName, entry := range m.SharesMap {
		if err := entry.RegistryShareAdd(entryName); err != nil {
			return fmt.Errorf("Failed to write Share [%s] ERROR: %w", entryName, err)
		}
		logger.DebugAppendF(Name, "[LOOP:ADDED %s]", entryName)
	}
	logger.DebugAppend(Name, "[LOOP:DONE]")
	return nil
}
