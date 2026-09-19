package app

import (
	"github.com/ghsemail/GeeGooAgent/internal/workflow/premarket"
)

func (a *App) wireKeyLevelFetcher() {
	if a == nil || a.Workflow == nil {
		return
	}
	if a.Config != nil && a.Config.DryRun {
		a.Workflow.SetKeyLevelFetcher(nil)
		return
	}
	if a.SignalAPI == nil {
		a.Workflow.SetKeyLevelFetcher(nil)
		return
	}
	a.Workflow.SetKeyLevelFetcher(premarket.SignalKeyLevelFetcher{Signal: a.SignalAPI})
}
