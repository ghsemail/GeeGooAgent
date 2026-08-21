package app

import (
	"github.com/ghsemail/GeeGooAgent/internal/agent"
)

func (a *App) wireSynthesizer() {
	if a == nil || a.Agent == nil {
		if a != nil && a.Workflow != nil {
			a.Workflow.SetSynthesizer(nil)
		}
		return
	}
	if a.SynthesisGateway == nil {
		a.Agent.SetReportSynthesizer(nil)
		if a.Workflow != nil {
			a.Workflow.SetSynthesizer(nil)
		}
		return
	}
	synth := agent.NewReportSynthesizer(a.SynthesisGateway, a.EffectiveSynthesisModel(), a.EventBus)
	a.Agent.SetReportSynthesizer(synth)
	if a.Workflow != nil {
		a.Workflow.SetSynthesizer(synth)
	}
}
