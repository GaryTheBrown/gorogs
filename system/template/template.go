package template

import (
	"gorogs/config"
	"gorogs/system"
)

const (
	Name       string                = "template"
	Type       system.SystemTypeEnum = system.Utility
	IsCritical bool                  = false
	AutoStart  bool                  = true
)

type Struct struct {
	sState system.SysStateEnum
}

func (_ *Struct) Name() string                        { return Name }
func (_ *Struct) Type() system.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                    { return IsCritical }
func (_ *Struct) AutoStart() bool                     { return AutoStart }
func (s *Struct) IsState(in system.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() system.SysStateEnum       { return s.sState }
func (_ *Struct) Dependencies() []string              { return nil }
func (_ *Struct) OrderAfter() []string                { return nil }
func (_ *Struct) Priority() int                       { return 100 }

var SystemInstance Struct

func init() {
	system.Register(&SystemInstance)
}

func (s *Struct) Config(cm config.ConfigMap) {}
func (s *Struct) Setup()                     {}
func (s *Struct) Start() error               { return nil }
func (s *Struct) Stop()                      {}
func (s *Struct) Healthcheck() error         { return nil }
