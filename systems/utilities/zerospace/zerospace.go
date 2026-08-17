package zerospace

import (
	"gorogs/logger"
	"gorogs/systems/systeminterface"
)

const (
	Name       = "ZeroSpace"
	Type       = systeminterface.Utility
	IsCritical = false
	AutoStart  = true
)

type Struct struct {
	sState systeminterface.SysStateEnum
}

func (_ *Struct) Name() string                               { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum       { return Type }
func (_ *Struct) IsCritical() bool                           { return IsCritical }
func (_ *Struct) AutoStart() bool                            { return AutoStart }
func (s *Struct) State(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) Setup() {
	logger.InfoContinue(s.Name(), "Setting Up Zero Space Overlay...")
	err := s.setupOverlay()
	if err != nil {
		logger.Fatal(s.Name(), "", err)
	}
	logger.InfoEnd(s.Name(), "[DONE]")
	s.sState = systeminterface.FINISHED

}

func (_ *Struct) Start() error       { return nil }
func (_ *Struct) Stop()              {}
func (_ *Struct) Healthcheck() error { return nil }
