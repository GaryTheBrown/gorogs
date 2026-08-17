package template

import "gorogs/systems/systeminterface"

type TemplateStruct struct {
	sState systeminterface.SysStateEnum
}

func (_ TemplateStruct) Name() string                                { return "template" }
func (_ TemplateStruct) Type() systeminterface.SystemTypeEnum        { return systeminterface.Utility }
func (_ TemplateStruct) IsCritical() bool                            { return false }
func (_ TemplateStruct) AutoStart() bool                             { return true }
func (t *TemplateStruct) State(in systeminterface.SysStateEnum) bool { return t.sState == in }

func (t *TemplateStruct) Setup()             {}
func (t *TemplateStruct) Start() error       { return nil }
func (t *TemplateStruct) Stop() error        { return nil }
func (t *TemplateStruct) Healthcheck() error { return nil }
