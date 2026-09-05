package playbookexec

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRunProbeSOP(t *testing.T) {
	session := runtime.NewSession()
	router := &Router{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			switch req.Name {
			case "search_code":
				return tools.Result{
					Status: tools.StatusOK,
					Data:   map[string]any{"items": []any{map[string]any{"code": "300308.SZ", "name": "中际旭创", "market": "CN"}}},
				}
			case "get_signal_combinations":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{map[string]any{
						"name":       "SAR信号配套MACD直方图趋势",
						"buy_signal": []any{map[string]any{"index": "SAR"}},
						"frequency":  "60m",
					}}},
				}
			case "probe_bot_signal_series":
				return tools.Result{
					Status:  tools.StatusOK,
					Summary: "buy_hits=3 sell_hits=2",
					Data:    map[string]any{"buy_hits": 3, "sell_hits": 2},
				}
			default:
				return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
			}
		},
	}
	result, ok := router.TryRunFromPlan(context.Background(), Input{
		Session:  session,
		UserText: "测一下中际旭创 SAR+MACD 买卖点",
		StepBase: 1,
	}, "signal_probe")
	if !ok || result.Failed {
		t.Fatalf("ok=%v failed=%v err=%s", ok, result.Failed, result.Error)
	}
	if !contains(result.AssistantText, "信号探测") {
		t.Fatalf("reply=%q", result.AssistantText)
	}
}

func TestEnrichAnalysisFromSession(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: "## 腾讯控股 00700.HK · 股价分析（daily）\n\n偏强"})
	plan := AnalysisRunPlan{}
	enrichAnalysisFromSession(&plan, session)
	if plan.StockQuery != "腾讯控股" && plan.StockQuery != "腾讯" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && stringIndex(s, sub) >= 0)
}

func TestAnalysisClarifyBeforeSearchCodeDone(t *testing.T) {
	session := runtime.NewSession()
	var events []string
	router := &Router{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			switch req.Name {
			case "search_code":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{
						map[string]any{"code": "00700.HK", "name": "腾讯控股", "market": "HK"},
						map[string]any{"code": "01698.HK", "name": "腾讯音乐-SW", "market": "HK"},
					}},
				}
			case "get_single_prompt_template":
				return tools.Result{
					Status: tools.StatusOK,
					Data:   map[string]any{"selected_prompt_id": "p1", "selected_prompt_name": "股价分析"},
				}
			case "get_mcp_analysis":
				return tools.Result{Status: tools.StatusOK, Summary: "ok", Data: map[string]any{"analysis_result": "偏强"}}
			default:
				return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
			}
		},
	}
	result, ok := router.TryRunFromPlan(context.Background(), Input{
		Session:  session,
		UserText: "分析一下腾讯",
		StepBase: 1,
		OnProgress: func(event string, _ map[string]any) {
			events = append(events, event)
		},
		ToolCtx: tools.Context{
			ClarifyFn: func(_ context.Context, _ string, choices []string) (string, bool) {
				return choices[0], true
			},
		},
	}, "stock_analysis")
	if !ok || result.Failed {
		t.Fatalf("ok=%v failed=%v err=%s events=%v", ok, result.Failed, result.Error, events)
	}
	startAt, clarifyAt, doneAt := -1, -1, -1
	for i, ev := range events {
		if ev == "tool_start" && startAt < 0 {
			startAt = i
		}
		if ev == "clarify" && clarifyAt < 0 {
			clarifyAt = i
		}
		if ev == "tool_done" && doneAt < 0 {
			doneAt = i
		}
	}
	if startAt < 0 || clarifyAt < 0 || doneAt < 0 {
		t.Fatalf("missing events=%v", events)
	}
	if !(startAt < clarifyAt && clarifyAt < doneAt) {
		t.Fatalf("order start=%d clarify=%d done=%d events=%v", startAt, clarifyAt, doneAt, events)
	}
}

func stringIndex(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
