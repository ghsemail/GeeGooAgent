package app

import (
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

// SignalDiagnoseLabConfig returns lab-api settings for signal-diagnose lab_mode.
func (a *App) SignalDiagnoseLabConfig() workflowchat.LabConfig {
	if a == nil || a.Config == nil {
		return workflowchat.LabConfig{}
	}
	return workflowchat.LabConfig{
		LabAPIURL: a.Config.LabAPIURL(),
		LabAPIKey: a.Config.LabAPIKey(),
	}
}
