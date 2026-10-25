package app

import (
	"context"
	"fmt"
	"strings"

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
