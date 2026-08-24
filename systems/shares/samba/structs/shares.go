package structs

import (
	"bufio"
	"bytes"
	"fmt"
	"gorogs/config"
	"gorogs/logger"
	"gorogs/systems/shares/samba/vars"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

type Share struct {
	Path    string
	Comment string
}

func NewShare(path, comment string) Share {
	commentPath := filepath.Join(path, ".comment")

	if f, err := os.Open(commentPath); err == nil {
		limitedReader := io.LimitReader(f, 60)
		scanner := bufio.NewScanner(limitedReader)
		if scanner.Scan() {
			firstLine := strings.TrimSpace(scanner.Text())
			if len(firstLine) > 0 {
				if len(firstLine) >= 60 {
					comment = firstLine[:57] + "..."
				} else {
					comment = firstLine
				}
			}
		}
		f.Close()
	}

	return Share{
		Path:    path,
		Comment: comment,
	}
}

var globalSettings = map[string]string{
	"browseable":            "yes",
	"readdir_attr:oneshot":  "yes",
	"smbd:asynclight":       "yes",
	"smbd:background_queue": "no",
}

func (s Share) ToINI(io io.Writer) {
	keyValueToLine("path", s.Path, SpaceBefore, io)
	keyValueToLine("comment", s.Comment, SpaceBefore, io)
	keyValueToLine("writeable", "no", SpaceBefore, io)
	keyValueToLine("guest ok", "yes", SpaceBefore, io)
	for key, value := range globalSettings {
		keyValueToLine(key, value, SpaceBefore, io)
	}

}

func (share Share) RegistryShareAdd(shareName string) error {
	cmdAdd := exec.Command(vars.NetPath, "conf", "addshare", shareName, share.Path, "writeable=no", "guest_ok=yes", share.Comment, "-s", vars.MasterConfigFile)
	if _, err := cmdAdd.CombinedOutput(); err != nil {
		return err
	}
	for key, value := range globalSettings {
		cmdBrowse := exec.Command(vars.NetPath, "conf", "setparm", shareName, key, value, "-s", vars.MasterConfigFile)
		if _, err := cmdBrowse.CombinedOutput(); err != nil {
			return err
		}
	}
	return nil
}

type ShareMap map[string]Share

func NewShareMap() ShareMap {
	shareMap := make(ShareMap)
	if err := shareMap.FirstFill(); err != nil {
		logger.Fatal("ShareMap", "Failed to fill in Shares list for first time. Error: %w", err)
	}

	return shareMap
}

func (s ShareMap) Count() int {
	return len(s)
}

func (s ShareMap) FirstFill() error {
	entries, err := os.ReadDir(config.ShareRoot)
	if err != nil {
		return err
	}
	validShareName := regexp.MustCompile(`^[a-zA-Z0-9_\-\.\s()]+$`)

	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "." || entry.Name() == ".." {
			continue
		}
		if !validShareName.MatchString(entry.Name()) {
			continue
		}
		s[entry.Name()] = NewShare(filepath.Join(config.ShareRoot, entry.Name()), "")
	}

	return nil
}
func (s ShareMap) GetSortedList() []string {
	var sorted []string
	for sectionName := range s {
		sorted = append(sorted, sectionName)
	}
	sort.Strings(sorted)
	return sorted
}

func (s ShareMap) ToINI(io io.Writer) {
	for _, shareName := range s.GetSortedList() {
		fmt.Fprintf(io, "[%s]\n", shareName)
		s[shareName].ToINI(io)
		fmt.Fprint(io, "\n")
	}
}

func (s ShareMap) ToByte() []byte {

	var buffer bytes.Buffer
	s.ToINI(&buffer)
	return buffer.Bytes()
}

func (s ShareMap) RegistryShareDelete(shareName string) error {
	cmdDel := exec.Command(vars.NetPath, "conf", "delshare", shareName, "-s", vars.MasterConfigFile)
	if _, err := cmdDel.CombinedOutput(); err != nil {
		return err
	}
	return nil
}
