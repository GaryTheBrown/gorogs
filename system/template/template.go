package template

import (
	"gorogs/config"
	"gorogs/system/systeminterface"
)

const (
	Name       string                         = "template"
	Type       systeminterface.SystemTypeEnum = systeminterface.Utility
	IsCritical bool                           = false
	AutoStart  bool                           = true
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
func (_ *Struct) Dependencies() []string                       { return nil }
func (_ *Struct) OrderAfter() []string                         { return nil }
func (_ *Struct) Priority() int                                { return 100 }

func init() {
	systeminterface.Register(&Struct{})
}

func (s *Struct) Config(cm config.ConfigMap) {}
func (s *Struct) Setup()                     {}
func (s *Struct) Start() error               { return nil }
func (s *Struct) Stop()                      {}
func (s *Struct) Healthcheck() error         { return nil }
