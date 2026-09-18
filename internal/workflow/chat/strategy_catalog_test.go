package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestResolveStrategyCatalogCombination(t *testing.T) {
	runTool := func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
		switch req.Name {
		case "get_signal_combinations":
			return tools.Result{
				Status: tools.StatusOK,
				Data: map[string]any{
					"items": []any{
						map[string]any{"name": "Macd4H节奏策略", "buy_signal": []any{map[string]any{"x": 1}}, "sell_signal": []any{map[string]any{"x": 1}}},
					},
				},
			}
		default:
			return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
		}
	}
	match, err := resolveStrategyCatalog(context.Background(), "Macd4H", tools.Context{}, runTool)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if match.Type != catalogTypeCombination {
		t.Fatalf("type=%s", match.Type)
	}
	if match.Label == "" {
		t.Fatal("expected label")
	}
}

func TestAssembleAgentCognitionDoc(t *testing.T) {
	flow := &Flow{
		CatalogLabel: "Macd4H",
		CatalogType:  catalogTypeCombination,
		CatalogRaw:   map[string]any{"name": "Macd4H", "signal_id": "sig-1", "brief": "4H MACD 节奏"},
	}
	body := fallbackCognitionBody(flow)
	doc := assembleAgentCognitionDoc(flow, body)
	if doc == "" || !containsAll(doc, "Macd4H", "Agent 策略认知", "一句话定位", "策略库原文", "doc_type: strategy_agent_cognition") {
		t.Fatalf("doc missing sections: %s", doc)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !containsSubstr(s, p) {
			return false
		}
	}
	return true
}

func containsSubstr(s, sub string) bool {
	return strings.Contains(s, sub)
}
