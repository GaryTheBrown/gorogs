package template

import "gorogs/systems/systeminterface"

const (
	Name       = "template"
	Type       = systeminterface.Utility
	IsCritical = false
	AutoStart  = true
)

type Struct struct {
	sState systeminterface.SysStateEnum
}

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Setup()             {}
func (s *Struct) Start() error       { return nil }
func (s *Struct) Stop()              {}
func (s *Struct) Healthcheck() error { return nil }
