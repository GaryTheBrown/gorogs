package system

import "gorogs/config"

type System interface {
	Name() string
	Type() SystemTypeEnum
	IsCritical() bool
	AutoStart() bool
	IsState(in SysStateEnum) bool
	GetState() SysStateEnum
	Dependencies() []string
	OrderAfter() []string
	Priority() int

	Config(cm config.ConfigMap)
	Setup()
	Start() error
	Stop()

	Healthcheck() error
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
