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

func (_ *Struct) Name() string                                 { return Name }
func (_ *Struct) Type() systeminterface.SystemTypeEnum         { return Type }
func (_ *Struct) IsCritical() bool                             { return IsCritical }
func (_ *Struct) AutoStart() bool                              { return AutoStart }
func (s *Struct) IsState(in systeminterface.SysStateEnum) bool { return s.sState == in }
func (s *Struct) GetState() systeminterface.SysStateEnum       { return s.sState }

func (s *Struct) Config() {
	// NOTHING TO CONFIGURE IN HERE
}

func (s *Struct) Setup() {
	logger.DebugContinue(Name, "Setting Up Zero Space Overlay...")
	err := s.setupOverlay()
	if err != nil {
		logger.Fatal(Name, "", err)
	}
	logger.DebugEnd(Name, "[DONE]")
	s.sState = systeminterface.FINISHED

}

func (_ *Struct) Start() error       { return nil }
func (_ *Struct) Stop()              {}
func (_ *Struct) Healthcheck() error { return nil }
