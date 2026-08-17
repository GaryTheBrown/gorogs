package systeminterface

type System interface {
	Name() string
	Type() SystemTypeEnum
	IsCritical() bool
	AutoStart() bool
	State(in SysStateEnum) bool

	Setup()
	Start() error
	Healthcheck() error
	Stop()
}

type SystemTypeEnum uint8

const (
	Share SystemTypeEnum = iota
	Beacon
	Utility
)

type SysStateEnum int8

const (
	FAILED SysStateEnum = iota - 1
	OFF
	SETUP
	STARTED
	STOPPED
	FINISHED
)
