package zerospace

import (
	"gorogs/logger"
	"gorogs/systems/systeminterface"
)

type ZeroSpaceStruct struct {
	sState systeminterface.SysStateEnum
}

func (m *ZeroSpaceStruct) Name() string                               { return "zerospace" }
func (m *ZeroSpaceStruct) Type() systeminterface.SystemTypeEnum       { return systeminterface.Utility }
func (m *ZeroSpaceStruct) IsCritical() bool                           { return false }
func (m *ZeroSpaceStruct) AutoStart() bool                            { return true }
func (m *ZeroSpaceStruct) State(in systeminterface.SysStateEnum) bool { return m.sState == in }
func (m *ZeroSpaceStruct) Setup() {
	logger.InfoContinue(m.Name(), "Setting Up Zero Space Overlay...")
	err := m.setupOverlay()
	if err != nil {
		logger.Fatal(m.Name(), "", err)
	}
	logger.InfoEnd(m.Name(), "[DONE]")
	m.sState = systeminterface.FINISHED

}

func (m *ZeroSpaceStruct) Start() error       { return nil }
func (m *ZeroSpaceStruct) Stop()              {}
func (m *ZeroSpaceStruct) Healthcheck() error { return nil }
