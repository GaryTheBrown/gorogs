package zerospace

import (
	"gorogs/logger"
	"gorogs/systems"
)

type ZeroSpaceStruct struct {
	sState systems.SysStateEnum
}

func (m *ZeroSpaceStruct) Name() string                       { return "zerospace" }
func (m *ZeroSpaceStruct) Type() systems.SystemTypeEnum       { return systems.Utility }
func (m *ZeroSpaceStruct) IsCritical() bool                   { return false }
func (m *ZeroSpaceStruct) AutoStart() bool                    { return true }
func (m *ZeroSpaceStruct) State(in systems.SysStateEnum) bool { return m.sState == in }
func (m *ZeroSpaceStruct) Setup() error {
	logger.InfoContinue(m.Name(), "Setting Up Zero Space Overlay...")
	err := m.setupOverlay()
	if err == nil {
		logger.InfoEnd(m.Name(), "[DONE]")
		m.sState = systems.FINISHED
	} else {
		logger.InfoEndF(m.Name(), "[ERROR] : %e", err)
	}

	return err

}

func (m *ZeroSpaceStruct) Start() error { return nil }
func (m *ZeroSpaceStruct) Stop() error  { return nil }

func (m *ZeroSpaceStruct) Healthcheck() error { return nil }
