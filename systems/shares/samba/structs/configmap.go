package structs

import (
	"bytes"
	"fmt"
	"gorogs/systems/shares/samba/vars"

	"io"
	"maps"
	"os"
	"sort"
)

const (
	GlobalName    = "global"
	ConfigBackend = "config backend"
	SpaceBefore   = "\t"
)

type ConfigSection map[string]string
type ConfigMap struct {
	sections map[string]ConfigSection
	mode     ModeEnum
}

func NewConfigMap(mode ModeEnum) ConfigMap {
	return ConfigMap{
		sections: make(map[string]ConfigSection),
		mode:     mode,
	}
}

func (cs ConfigSection) ToINI(io io.Writer) {
	var innerKeys []string
	hasConfigBackend := false

	for k := range cs {
		if k == ConfigBackend {
			hasConfigBackend = true
			continue
		}
		innerKeys = append(innerKeys, k)
	}
	sort.Strings(innerKeys)

	if hasConfigBackend {
		keyValueToLine(ConfigBackend, cs[ConfigBackend], SpaceBefore, io)
	}

	for _, k := range innerKeys {
		keyValueToLine(k, cs[k], SpaceBefore, io)
	}
	fmt.Fprint(io, "\n")
}

func (cs ConfigSection) ToByte() []byte {
	if len(cs) == 0 {
		return nil
	}

	var buffer bytes.Buffer
	cs.ToINI(&buffer)
	return buffer.Bytes()
}

func (cs ConfigSection) SectionMerge(src ConfigSection) {
	maps.Copy(cs, src)

}

func (cm ConfigMap) ToINI(io io.Writer) {
	if globalMap, hasGlobal := cm.sections[GlobalName]; hasGlobal {
		fmt.Fprintf(io, "[%s]\n", GlobalName)
		globalMap.ToINI(io)
	} else {
		// logger.Fatal()
	}

	var sortedSubsections []string
	for sectionName := range cm.sections {
		if sectionName != GlobalName {
			sortedSubsections = append(sortedSubsections, sectionName)
		}
	}
	sort.Strings(sortedSubsections)

	for _, section := range sortedSubsections {
		fmt.Fprintf(io, "[%s]\n", section)
		cm.sections[section].ToINI(io)
	}

	switch cm.mode {
	case ModeFile:
		keyValueToLine("include", vars.ShareConfigFile, "", io)
	case ModeMixed:
		keyValueToLine("include", "registry", "", io)
	}

}

func (cm ConfigMap) ToByte() []byte {
	if len(cm.sections) == 0 {
		return nil
	}

	var buffer bytes.Buffer
	cm.ToINI(&buffer)
	return buffer.Bytes()
}
func (cm ConfigMap) ToFile(location string) error {
	file, err := os.Create(location)
	if err != nil {
		return err
	}
	defer file.Close()
	cm.ToINI(file)
	return nil
}

func (cm ConfigMap) Merge(src ConfigMap) {
	for section, innerMap := range src.sections {
		if _, exists := cm.sections[section]; !exists {
			cm.sections[section] = make(ConfigSection)
		}
		maps.Copy(cm.sections[section], innerMap)
	}
}

func (cm ConfigMap) SetSection(name string, section ConfigSection) {
	cm.sections[name] = section
}

func (cm ConfigMap) SectionMerge(section string, src ConfigSection) {
	if _, exists := cm.sections[section]; !exists {
		return
	}
	maps.Copy(cm.sections[section], src)

}

func (cm ConfigMap) GetSection(name string) ConfigSection {
	return cm.sections[name]
}

func (cm ConfigMap) Count() int {
	return len(cm.sections)
}
