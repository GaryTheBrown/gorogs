package structs

import "strings"

type ModeEnum uint8

const (
	ModeNOTSET ModeEnum = iota
	ModeFile
	ModeMixed
	ModeRegistry
)

var (
	modeStringMap map[ModeEnum]string = map[ModeEnum]string{
		ModeNOTSET:   "NOTSET",
		ModeFile:     "File",
		ModeMixed:    "Mixed",
		ModeRegistry: "Registry",
	}
)

func ModeToString(mode ModeEnum) string {
	return modeStringMap[mode]
}

func StringToMode(mode string) ModeEnum {
	for k, v := range modeStringMap {
		if strings.EqualFold(v, mode) {
			return k
		}
	}
	return ModeNOTSET
}
