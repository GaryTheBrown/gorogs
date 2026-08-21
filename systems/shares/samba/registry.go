package samba

import (
	"bufio"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func (s *Struct) injectAllSharesToRegistry() {
	s.injectGlobalSettingsToRegistry()

	entries, err := os.ReadDir(config.ShareRoot)
	if err != nil {
		logger.ErrorF(s.Name(), "Failed to read storage source root folder: %v", err, err.Error())
		return
	}

	validShareName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s()]+$`)
	count := 0

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "." || entry.Name() == ".." || entry.Name() == "nfs" || entry.Name() == "ganesha" {
			continue
		}
		if !validShareName.MatchString(entry.Name()) {
			continue
		}

		fullPath := filepath.Join(config.ShareRoot, entry.Name())

		if err := s.executeRegistryAdd(entry.Name(), fullPath); err == nil {
			count++
		}
	}

	logger.InfoF(s.Name(), "Successfully synchronized global configuration blocks and [%d] dynamic shares directly into local registry.", count)

	s.dumpRegistryConfigurationToLog()
}

func (s *Struct) injectGlobalSettingsToRegistry() {
	globals := map[string]string{
		"workgroup":              "WORKGROUP",
		"server string":          "Read only Share",
		"netbios name":           strings.ToUpper(config.Hostname),
		"security":               "user",
		"map to guest":           "bad user",
		"usershare allow guests": "yes",
		"load printers":          "no",
		"printcap name":          "/dev/null",
		"log file":               "/dev/null",
		"max log size":           "0",
		"log level":              "0",
		"veto files":             "/.*/",
		"guest account":          "smbguest",
	}

	if config.IsEnabled("netbios") {
		globals["dns proxy"] = "no"
		globals["hostname lookups"] = "no"
	}

	for parameter, val := range globals {
		cmdGlobal := exec.Command(netPath, "conf", "setparm", "global", parameter, val, "-s", masterConfigPath)
		output, err := cmdGlobal.CombinedOutput()
		if err != nil {
			logger.ErrorF(s.Name(), "Failed to write local global parameter [%s]: %s ERROR: %v", err, parameter, strings.TrimSpace(string(output)), err.Error())
		}
		logger.DebugF(s.Name(), "[NET]initialize global parameter [%s]: %s", parameter, strings.TrimSpace(string(output)))
	}

	cmdDelVFS := exec.Command(netPath, "conf", "delparm", "global", "vfs objects", "-s", masterConfigPath)
	_ = cmdDelVFS.Run()
}

func (s *Struct) dumpRegistryConfigurationToLog() {
	logger.Info(s.Name(), "[DIAGNOSTIC] Querying raw registry backend database configurations via net conf list...")

	cmdDump := exec.Command(netPath, "conf", "list", "-s", masterConfigPath)
	output, err := cmdDump.CombinedOutput()
	if err != nil {
		logger.ErrorF(s.Name(), "Diagnostic database configuration lookup failed: %s ERROR: %v", err, strings.TrimSpace(string(output)), err.Error())
		return
	}

	rawTextOutput := string(output)
	if len(strings.TrimSpace(rawTextOutput)) == 0 {
		logger.Info(s.Name(), "[DIAGNOSTIC ALERT] The active registry database file is physically EMPTY! Samba sees 0 parameters.")
		return
	}

	scanner := bufio.NewScanner(strings.NewReader(rawTextOutput))
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.TrimSpace(line)) > 0 {
			logger.InfoF(s.Name(), "[DB-DUMP] %s", line)
		}
	}
}

func (s *Struct) executeRegistryAdd(shareName, physicalPath string) error {
	commentPath := filepath.Join(physicalPath, ".comment")
	shareComment := "Read-only Media Stream"

	if f, err := os.Open(commentPath); err == nil {
		limitedReader := io.LimitReader(f, 60)
		scanner := bufio.NewScanner(limitedReader)
		if scanner.Scan() {
			firstLine := strings.TrimSpace(scanner.Text())
			if len(firstLine) > 0 {
				if len(firstLine) >= 60 {
					shareComment = firstLine[:57] + "..."
				} else {
					shareComment = firstLine
				}
			}
		}
		f.Close()
	}

	cmdAdd := exec.Command(netPath, "conf", "addshare", shareName, physicalPath, "writeable=no", "guest_ok=yes", shareComment, "-s", masterConfigPath)
	output, err := cmdAdd.CombinedOutput()
	if err != nil {
		logger.ErrorF(s.Name(), "Failed to execute addshare on target [%s]: %s ERROR: %v", err, shareName, strings.TrimSpace(string(output)), err.Error())
		return fmt.Errorf("local share creation failed: %w (Output: %s)", err, strings.TrimSpace(string(output)))
	}
	logger.DebugF(s.Name(), "[NET]addshare executed successfully on target [%s]: %s", shareName, strings.TrimSpace(string(output)))

	cmdBrowse := exec.Command(netPath, "conf", "setparm", shareName, "browseable", "yes", "-s", masterConfigPath)
	if outBrowse, errBrowse := cmdBrowse.CombinedOutput(); errBrowse != nil {
		logger.ErrorF(s.Name(), "Failed to execute browseable parameter set on target [%s]: %s ERROR: %v", errBrowse, shareName, strings.TrimSpace(string(outBrowse)), errBrowse.Error())
	}

	return nil
}

func (s *Struct) executeRegistryDelete(shareName string) error {
	cmdDel := exec.Command(netPath, "conf", "delshare", shareName, "-s", masterConfigPath)
	output, err := cmdDel.CombinedOutput()
	if err != nil {
		logger.ErrorF(s.Name(), "Failed to execute delshare on target [%s]: %s ERROR: %v", err, shareName, strings.TrimSpace(string(output)), err.Error())
		return fmt.Errorf("local share deletion failed: %w (Output: %s)", err, strings.TrimSpace(string(output)))
	}
	logger.DebugF(s.Name(), "[NET]delshare executed successfully on target [%s]: %s", shareName, strings.TrimSpace(string(output)))

	return nil
}
