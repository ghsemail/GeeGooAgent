package chat

import (
	"context"
	"strings"
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
						"buy_hits": 15, "sell_hits": 16, "bar_count": 5,
						"bars": []any{
							map[string]any{"time": "2026-06-01T10:00:00", "close": 100.0},
							map[string]any{"time": "2026-06-02T10:00:00", "close": 105.0},
							map[string]any{"time": "2026-06-03T10:00:00", "close": 103.0},
							map[string]any{"time": "2026-06-04T10:00:00", "close": 98.0},
							map[string]any{"time": "2026-06-05T10:00:00", "close": 95.0},
						},
						"buy_merged":  []any{1, 0, 0, 0, 0},
						"sell_merged": []any{0, 0, 0, -1, 0},
					},
				}
			case "save_strategy_knowledge":
				return tools.Result{
					Status:  tools.StatusOK,
					Summary: "saved",
					Data:    map[string]any{"knowledge_id": "kb-test-1", "title": "SAR · 腾讯控股 · 信号诊断"},
				}
			case "get_key_levels":
				return tools.Result{
					Status: tools.StatusOK,
					Summary: "mock levels",
					Data: map[string]any{
						"judgment": map[string]any{
							"summary": "mock",
							"support":    map[string]any{"low": 50.0, "high": 52.0, "center": 51.0},
							"resistance": map[string]any{"low": 110.0, "high": 112.0, "center": 111.0},
						},
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
	if !IsSignalDiagnoseIntent("诊断 SAR信号配套MACD直方图趋势 · 五粮液 (000858.SZ)") {
		t.Fatal("expected wuliangye diagnose intent")
	}
	if IsSignalDiagnoseIntent("读取 Macd4H 策略") {
		t.Fatal("strategy dev should not match signal diagnose")
	}
}

func TestParseSignalDiagnoseMessageComposedPanelStyle(t *testing.T) {
	text := "信号诊断：帮我用 SAR 信号作为买入信号，用 4小时MACD市场节奏 信号作为卖出信号，关闭阻力支撑熔断，测试一下腾讯控股（00700.HK）。"
	strategy, stock := parseSignalDiagnoseMessage(text)
	if strategy != "买:SAR · 卖:4小时MACD市场节奏" {
		t.Fatalf("strategy=%q", strategy)
	}
	if !strings.Contains(stock, "00700.HK") {
		t.Fatalf("stock=%q", stock)
	}
}

func TestParseSignalDiagnoseMessageWuliangye(t *testing.T) {
	strategy, stock := parseSignalDiagnoseMessage("诊断 SAR信号配套MACD直方图趋势 · 五粮液 (000858.SZ)")
	if strategy != "SAR信号配套MACD直方图趋势" {
		t.Fatalf("strategy=%q", strategy)
	}
	if stock != "五粮液 (000858.SZ)" {
		t.Fatalf("stock=%q", stock)
	}
}

func TestShouldStartSignalDiagnoseFlowAfterPause(t *testing.T) {
	paused := &Flow{Template: SkillSignalDiagnose, Status: StatusPausedFailed, Phase: PhaseResolveSymbol}
	if !ShouldStartSignalDiagnoseFlow("诊断 SAR · 000858.SZ", paused) {
		t.Fatal("expected restart after paused_failed")
	}
	running := &Flow{Template: SkillSignalDiagnose, Status: StatusRunning, Phase: PhaseRunProbe}
	if ShouldStartSignalDiagnoseFlow("诊断 SAR · 000858.SZ", running) {
		t.Fatal("should not restart mid-run")
	}
}

func TestEvalWorkflowSignalDiagnoseComposedPanelNoClarify(t *testing.T) {
	if !IsSignalDiagnoseIntent("信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 信号作为卖出信号，开启阻力支撑熔断，测试一下腾讯控股（00700.HK）。") {
		t.Fatal("composed panel message should start signal_diagnose")
	}
	runner := signalDiagnoseTestRunner(t)
	clarifyCalls := 0
	var toolsCalled []string
	base := runner.RunTool
	runner.RunTool = func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		toolsCalled = append(toolsCalled, req.Name)
		if req.Name == "clarify" {
			clarifyCalls++
		}
		return base(ctx, req, tc)
	}
	msg := "信号诊断：帮我用 SAR 信号作为买入信号，用 SAR 信号作为卖出信号，开启阻力支撑熔断，测试一下腾讯控股（00700.HK）。"
	session := runtime.NewSession()
	session.PendingSignalDiagnoseOpts = &runtime.SignalDiagnoseOpts{
		UseKeyLevelEpisodeStop: true,
		KeyBreakMode:           KeyBreakModeResistHigh,
		PanelStrategyLabel:     "买:SAR · 卖:SAR",
		Frequency:              "60m",
		MonthsBack:             3,
		BuySignal: []any{
			map[string]any{"index": "SAR", "type": "signal"},
		},
		SellSignal: []any{
			map[string]any{"index": "SAR", "type": "signal"},
		},
	}
	toolCtx := tools.Context{
		ClarifyFn: func(context.Context, string, []string) (string, bool) {
			clarifyCalls++
			t.Fatal("ClarifyFn must not run for panel-style signal_diagnose eval")
			return "", false
		},
	}
	result, handled := runner.RunTurn(context.Background(), session, msg, toolCtx, 1)
	if !handled || result.Failed {
		t.Fatalf("handled=%v failed=%v err=%s", handled, result.Failed, result.Error)
	}
	if clarifyCalls > 0 {
		t.Fatalf("clarify invoked %d times", clarifyCalls)
	}
	for _, name := range toolsCalled {
		if name == "clarify" {
			t.Fatalf("clarify tool called during workflow")
		}
	}
	if !contains(result.AssistantText, "Episode") && !contains(result.AssistantText, "命中率") {
		t.Fatalf("missing eval report: %q", result.AssistantText)
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
	if !contains(result.AssistantText, "Episode") && !contains(result.AssistantText, "命中率") {
		t.Fatalf("missing eval section: %q", result.AssistantText)
	}
	if !contains(result.AssistantText, "持有K线") {
		t.Fatalf("missing episode detail: %q", result.AssistantText)
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
