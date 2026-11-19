package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// StrategyDevJevConfig wires Jev to GeeGooData Decision API: data-node base URL + service Bearer, model from ops catalog.
func (a *App) StrategyDevJevConfig() strategydev.JevConfig {
	if a == nil || a.Config == nil {
		return strategydev.JevConfig{}
	}
	a.RefreshOpsDecision(false)
	dec := a.Config.ResolvedDecision()
	node := a.Config.ResolvedDecisionDataNode()
	bearer := strings.TrimSpace(node.Bearer)
	if bearer == "" {
		return strategydev.JevConfig{}
	}
	model := dec.Model
	if strings.TrimSpace(model) == "" {
		model = config.DefaultDecisionModel
	}
	// Lab lanes call GeeGooData /v1/decision/choice with the data-node bearer. Ops catalog supplies
	// the model id when configured; batch/cron paths must still work when only data tokens are set.
	if !dec.Configured {
		return strategydev.JevConfig{
			BaseURL: node.BaseURL,
			APIKey:  bearer,
			Model:   model,
		}
	}
	return strategydev.JevConfig{
		BaseURL: node.BaseURL,
		APIKey:  bearer,
		Model:   model,
	}
}

// NewStrategyDevJevClient builds Jev for Lab: System One direction via Data, reason via synthesis LLM.
func (a *App) NewStrategyDevJevClient() *strategydev.JevClient {
	j := strategydev.NewJevClient(a.StrategyDevJevConfig())
	a.attachStrategyDevJevReasonLLM(j)
	return j
}

func (a *App) attachStrategyDevJevReasonLLM(j *strategydev.JevClient) {
	if a == nil || j == nil || a.SynthesisGateway == nil {
		return
	}
	gw := a.SynthesisGateway
	j.SetReasonChat(func(ctx context.Context, system, user string) (string, error) {
		resp, err := gw.ChatSynthesis(ctx, []llm.Message{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		}, func(r *llm.Response) error {
			if r == nil || strings.TrimSpace(r.Content) == "" {
				return fmt.Errorf("empty synthesis content")
			}
			return nil
		})
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(resp.Content), nil
	})
}
