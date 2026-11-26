package app

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

type countingProvider struct {
	name, reply string
	calls       *[]string
}

func (p countingProvider) Model() string { return p.name }
func (p countingProvider) Chat(context.Context, []llm.Message, []llm.ToolSchema, float64, int) (*llm.Response, error) {
	*p.calls = append(*p.calls, p.name)
	return &llm.Response{Content: p.reply}, nil
}

func TestMacroNewsLLMUsesAtMostOneFallback(t *testing.T) {
	var calls []string
	gateway := llm.NewGateway(countingProvider{"primary", "bad", &calls}, llm.GatewayConfig{MaxRetries: 3, RetryWait: 1})
	gateway.SetFallbacks([]llm.Provider{
		countingProvider{"second", "bad", &calls}, countingProvider{"third", "good", &calls},
	})
	a := &App{SynthesisGateway: gateway}
	accept := func(content string) error {
		if content != "good" {
			return fmt.Errorf("unusable")
		}
		return nil
	}
	if _, err := a.MacroNewsLLM().Complete(context.Background(), "s", "u", accept); err == nil {
		t.Fatal("the third model must not be tried")
	}
	if fmt.Sprint(calls) != "[primary second]" {
		t.Fatalf("calls=%v", calls)
	}
	// The shared gateway keeps its own fallback list.
	calls = nil
	if _, err := gateway.ChatSynthesis(context.Background(), nil, func(r *llm.Response) error { return accept(r.Content) }); err != nil {
		t.Fatal(err)
	}
	if (&App{}).MacroNewsLLM() != nil {
		t.Fatal("no gateway must yield no LLM")
	}
}
