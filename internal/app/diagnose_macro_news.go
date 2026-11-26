package app

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// macroNewsGatewayLLM runs diagnose_macro_news prompts on the synthesis gateway,
// limited to the primary model and one fallback.
type macroNewsGatewayLLM struct{ gateway *llm.Gateway }

func (m macroNewsGatewayLLM) Complete(ctx context.Context, system, user string, accept func(string) error) (string, error) {
	resp, err := m.gateway.ChatSynthesis(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: system},
		{Role: llm.RoleUser, Content: user},
	}, func(r *llm.Response) error {
		if r == nil || strings.TrimSpace(r.Content) == "" {
			return fmt.Errorf("empty model reply")
		}
		return accept(r.Content)
	})
	if err != nil {
		return "", err
	}
	return resp.Usage.Model, nil
}

// MacroNewsLLM is nil while no synthesis model is configured.
func (a *App) MacroNewsLLM() diagnoseskill.MacroNewsLLM {
	if a == nil || a.SynthesisGateway == nil {
		return nil
	}
	return macroNewsGatewayLLM{gateway: a.SynthesisGateway.WithSingleFallback()}
}
