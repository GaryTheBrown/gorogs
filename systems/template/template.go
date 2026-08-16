package template

import "gorogs/systems"

type TemplateStruct struct {
	sState systems.SysStateEnum
}

func (_ TemplateStruct) Name() string                        { return "template" }
func (_ TemplateStruct) Type() systems.SystemTypeEnum        { return systems.Utility }
func (_ TemplateStruct) IsCritical() bool                    { return false }
func (_ TemplateStruct) AutoStart() bool                     { return true }
func (t *TemplateStruct) State(in systems.SysStateEnum) bool { return t.sState == in }

func (t *TemplateStruct) Setup() error       { return nil }
func (t *TemplateStruct) Start() error       { return nil }
func (t *TemplateStruct) Stop() error        { return nil }
func (t *TemplateStruct) Healthcheck() error { return nil }
