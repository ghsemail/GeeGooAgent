package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func signalDiagnoseTestRunner(t *testing.T) *Runner {
	t.Helper()
	return &Runner{
		RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
			switch req.Name {
			case "get_signal_combinations", "get_custom_signal_for_skill", "get_custom_strategy_definitions":
				return tools.Result{Status: tools.StatusError, Summary: "skip"}
			case "get_index_signals":
				return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
					map[string]any{
						"name": "SAR", "index": "SAR", "frequency": "60m",
						"buy_signal": []any{map[string]any{
							"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
						}},
						"sell_signal": []any{map[string]any{
							"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
						}},
					},
				}}}
			case "search_code":
				return tools.Result{
					Status: tools.StatusOK,
					Data: map[string]any{"items": []any{map[string]any{
						"code": "00700.HK", "name": "腾讯控股", "market": "HK",
					}}},
				}
			case "probe_bot_signal_series":
				return tools.Result{
					Status:  tools.StatusOK,
					Summary: "probe ok",
					Data: map[string]any{
						"buy_hits": 15, "sell_hits": 16, "bar_count": 462,
					},
				}
			case "diagnose_bot_signal_series":
				return tools.Result{
					Status:  tools.StatusOK,
					Summary: "diagnose ok",
					Data: map[string]any{
						"verdict": "has_triggers",
						"summary": "买入 15 次，卖出 16 次",
						"hits":    map[string]any{"buy": 15, "sell": 16},
						"buy_rules": []any{map[string]any{
							"index": "SAR", "type": "signal",
							"param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
							"algorithm": "SAR 方向翻转", "trigger_count": 15,
							"last_bar": map[string]any{"signal": 0, "reason": "SAR signal=hold"},
						}},
					},
				}
			default:
				return tools.Result{Status: tools.StatusError, Summary: "unexpected " + req.Name}
			}
		},
	}
}

func TestIsSignalDiagnoseIntent(t *testing.T) {
	if !IsSignalDiagnoseIntent("诊断 SAR 策略 · 腾讯") {
		t.Fatal("expected signal diagnose intent")
	}
	if IsSignalDiagnoseIntent("读取 Macd4H 策略") {
		t.Fatal("strategy dev should not match signal diagnose")
	}
}

func TestSignalDiagnoseFlowCompletes(t *testing.T) {
	runner := signalDiagnoseTestRunner(t)
	runner.RunTool = func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		if req.Name == "get_signal_combinations" {
			return tools.Result{Status: tools.StatusError, Summary: "not found"}
		}
		if req.Name == "get_index_signals" {
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{"items": []any{
				map[string]any{
					"name": "SAR", "index": "SAR", "frequency": "60m",
					"buy_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
					"sell_signal": []any{map[string]any{
						"index": "SAR", "type": "signal", "param": map[string]any{"acceleration": 0.02, "maximum": 0.2},
					}},
				},
			}}}
		}
		return signalDiagnoseTestRunner(t).RunTool(ctx, req, tc)
	}
	session := runtime.NewSession()
	result, handled := runner.RunTurn(context.Background(), session, "诊断 SAR 策略 · 腾讯", tools.Context{}, 1)
	if !handled || result.Failed {
		t.Fatalf("handled=%v failed=%v err=%s text=%s", handled, result.Failed, result.Error, result.AssistantText)
	}
	if !contains(result.AssistantText, "信号诊断") {
		t.Fatalf("reply=%q", result.AssistantText)
	}
	if !contains(result.AssistantText, "15") || !contains(result.AssistantText, "SAR") {
		t.Fatalf("missing probe/diagnose details: %q", result.AssistantText)
	}
	if LoadFromSession(session) != nil {
		t.Fatal("expected flow cleared after completion")
	}
}

func TestFormatSignalDiagnoseMessage(t *testing.T) {
	got := FormatSignalDiagnoseMessage("SAR", "腾讯")
	if got != "诊断 SAR 策略 · 腾讯" {
		t.Fatalf("message = %q", got)
	}
}
