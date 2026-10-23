package app

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// StrategyDevJevConfig wires Jev to GeeGooData Decision API: data-node base URL + service Bearer, model from ops catalog.
func (a *App) StrategyDevJevConfig() strategydev.JevConfig {
	if a == nil || a.Config == nil {
		return strategydev.JevConfig{}
	}
	a.RefreshOpsDecision(false)
	dec := a.Config.ResolvedDecision()
	if !dec.Configured {
		return strategydev.JevConfig{}
	}
	node := a.Config.ResolvedDecisionDataNode()
	bearer := strings.TrimSpace(node.Bearer)
	if bearer == "" {
		return strategydev.JevConfig{}
	}
	return strategydev.JevConfig{
		BaseURL: node.BaseURL,
		APIKey:  bearer,
		Model:   dec.Model,
	}
}
