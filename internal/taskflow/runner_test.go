package taskflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func testRunner(t *testing.T) *Runner {
	t.Helper()
	return &Runner{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			switch req.Name {
			case "search_code":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{map[string]any{
						"code": "00700.HK", "name": "腾讯控股", "market": "HK",
					}}},
				}
			case "get_signal_combinations":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{
						map[string]any{
							"name": "Macd4HRhythm", "frequency": "60m",
							"buy_signal": []any{map[string]any{"index": "MACD"}},
							"sell_signal": []any{map[string]any{"index": "MACD"}},
						},
						map[string]any{
							"name": "MACDResonance", "frequency": "15m",
							"buy_signal": []any{map[string]any{"index": "MACD"}},
							"sell_signal": []any{map[string]any{"index": "MACD"}},
						},
					}},
				}
			case "probe_bot_signal_series":
				label := fmt.Sprint(req.Arguments["buy_signal"])
				return tools.Result{
					Status:  tools.StatusOK,
					Summary: "buy_hits=2 sell_hits=1",
					Data:    map[string]any{"buy_hits": 2, "sell_hits": 1, "label": label},
				}
			default:
				return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
			}
		},
	}
}

func TestIsContinueAllIntent(t *testing.T) {
	if !IsContinueAllIntent("你挨个跑一下") {
		t.Fatal("expected continue-all intent")
	}
}

func TestMultiStrategyFlowCompletes(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "腾讯"})
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: "已选 Macd4H"})
	result, handled := testRunner(t).RunTurn(context.Background(), session, "你挨个跑一下", tools.Context{}, 1)
	if !handled || result.Failed {
		t.Fatalf("handled=%v failed=%v err=%s text=%s", handled, result.Failed, result.Error, result.AssistantText)
	}
	if !contains(result.AssistantText, "多策略信号对比") {
		t.Fatalf("reply=%q", result.AssistantText)
	}
	if LoadFromSession(session) != nil {
		t.Fatal("expected flow cleared after completion")
	}
}

func TestResumePausedFlow(t *testing.T) {
	flow := &Flow{
		RunID: "flow-test", Template: TemplateMultiStrategyCompare,
		Status: StatusPausedFailed, Phase: PhaseResolveSymbol,
		StockQuery: "腾讯",
	}
	session := runtime.NewSession()
	SaveToSession(session, flow)
	result, handled := testRunner(t).RunTurn(context.Background(), session, "继续", tools.Context{}, 1)
	if !handled {
		t.Fatal("expected handled resume")
	}
	if result.Failed && !contains(result.AssistantText, "暂停") {
		t.Fatalf("unexpected failure: %s", result.Error)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || len(s) > 0 && stringIndex(s, sub) >= 0))
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
