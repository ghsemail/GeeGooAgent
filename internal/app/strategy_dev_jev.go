package app

import (
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// StrategyDevJevConfig returns Jev (System One) settings from ops catalog decision model.
func (a *App) StrategyDevJevConfig() strategydev.JevConfig {
	if a == nil || a.Config == nil {
		return strategydev.JevConfig{}
	}
	a.RefreshOpsDecision(false)
	dec := a.Config.ResolvedDecision()
	if !dec.Configured {
		return strategydev.JevConfig{}
	}
	return strategydev.JevConfig{
		BaseURL: dec.BaseURL,
		APIKey:  dec.TokenKey,
		Model:   dec.Model,
	}
}
