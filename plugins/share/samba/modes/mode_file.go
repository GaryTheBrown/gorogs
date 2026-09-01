package modes

import (
	"fmt"
	"gorogs/logger"
	"gorogs/plugins/share/samba/structs"
	"gorogs/plugins/share/samba/vars"
	"os"
	"syscall"
	"time"
)

type ModeFile struct {
	ConfigMap structs.ConfigMap
	SharesMap structs.ShareMap
}

func (m *ModeFile) Setup() error {
	logger.Debug(Name, "[WRITE CONFIG FILE]")
	return m.writeConfigFile()
}

func (m *ModeFile) writeConfigFile() error {
	file, err := os.Create(vars.ShareConfigFile)
	if err != nil {
		return err
	}
	defer file.Close()
	file.Write(m.ConfigMap.ToByte())
	file.WriteString("\n\n")
	file.Write(m.SharesMap.ToByte())
	return nil
}

func (m *ModeFile) NotifyCreate(shareName string, path string) error {
	m.SharesMap[shareName] = structs.NewShare(path)
	return m.notify()
}

func (m *ModeFile) NotifyRemove(shareName string) error {
	if _, exists := m.SharesMap[shareName]; exists {
		delete(m.SharesMap, shareName)
		return m.notify()
	}
	return fmt.Errorf("Share Not Found in List to Remove: %s", shareName)
}

func (m *ModeFile) NotifyCommentUpdate(shareName, comment string) error {
	if share, exists := m.SharesMap[shareName]; exists {
		share.Comment = comment
		m.SharesMap[shareName] = share
		return m.notify()
	}
	return fmt.Errorf("Share Not Found in List to Update Comment: %s", shareName)
}

var debounceTimer *time.Timer

const debounceDuration = 250 * time.Millisecond

func (m *ModeFile) notify() error {
	if debounceTimer != nil {
		debounceTimer.Stop()
	}
	var returnError error
	debounceTimer = time.AfterFunc(debounceDuration, func() {
		if err := m.writeConfigFile(); err != nil {
			returnError = fmt.Errorf("Failed to write to share text block: %w", err)
			return
		}

		if vars.Cmd == nil || vars.Cmd.Process == nil {
			return
		}
		if err := vars.Cmd.Process.Signal(syscall.SIGHUP); err != nil {
			returnError = fmt.Errorf("Failed hot-reload samba process: %w", err)
		}
	})
	return returnError
}
