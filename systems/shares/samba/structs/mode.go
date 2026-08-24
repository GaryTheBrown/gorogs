package structs

type ModeEnum uint8

const (
	ModeNOTSET ModeEnum = iota
	ModeFile
	ModeMixed
	ModeRegistry
)

var modeStringMap = map[ModeEnum]string{
	ModeNOTSET:   "NOTSET",
	ModeFile:     "File",
	ModeMixed:    "Mixed",
	ModeRegistry: "Registry",
}

func ModeToString(mode ModeEnum) string {
	return modeStringMap[mode]
}
