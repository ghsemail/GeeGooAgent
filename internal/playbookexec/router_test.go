package playbookexec

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestRouteBacktestRun(t *testing.T) {
	skills := []string{"strategy-backtest-run", "strategy-backtest"}
	if p, ok := Route(skills, "帮我回测一下 sar 信号加 macd 在小米", nil); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v", p, ok)
	}
	if _, ok := Route(skills, "帮我做 dca 定投回测", nil); ok {
		t.Fatal("dca bypass should not route")
	}
	if _, ok := Route([]string{"strategy-signal-probe"}, "帮我回测一下", nil); !ok {
		t.Fatal("backtest intent should route even without backtest playbook skill match")
	}
}

func TestRouteBacktestWithoutMatchedSkills(t *testing.T) {
	msg := "用现成的来回测，不要新建"
	if procedural.BacktestRunIntent(msg) {
		t.Fatal("continue phrasing should not count as backtest run intent")
	}
	if p, ok := Route(nil, msg, nil); ok {
		t.Fatalf("continue without session should not route, got %q", p)
	}
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 小米"})
	if p, ok := Route(nil, msg, session); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v want playbook with session context", p, ok)
	}
}

func TestRouteBacktestContinueWithSession(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 小米"})
	if p, ok := Route(nil, "就用刚才那套", session); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v", p, ok)
	}
}

func TestHeuristicBacktestPlan(t *testing.T) {
	plan := heuristicBacktestPlan("帮我测试一下帮我回测一下sar信号加macd趋势在小米")
	if plan.StockQuery != "小米" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.SignalKind != "combination" {
		t.Fatalf("kind=%q", plan.SignalKind)
	}
}

func TestHeuristicBacktestPlanAppleMACDStrategy(t *testing.T) {
	msg := "请帮我用「4小时MACD市场节奏」策略测试一下Apple（AAPL · 美股）。"
	plan := heuristicBacktestPlan(msg)
	if plan.StockQuery != "AAPL" {
		t.Fatalf("stock=%q want AAPL", plan.StockQuery)
	}
	if plan.SignalQuery != "4小时MACD市场节奏" {
		t.Fatalf("signal=%q want named strategy", plan.SignalQuery)
	}
	if plan.SignalKind != "combination" {
		t.Fatalf("kind=%q", plan.SignalKind)
	}
}

func TestExtractStockQuerySkipsIndicatorTokens(t *testing.T) {
	if q := extractStockQuery("回测 MACD 组合在 Tesla"); q != "TESLA" {
		t.Fatalf("stock=%q want TESLA", q)
	}
	if q := extractStockQuery("「4小时MACD市场节奏」Apple（AAPL）"); q != "AAPL" {
		t.Fatalf("stock=%q", q)
	}
}

func TestEnrichPlanFromSession(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 在小米"})
	plan := BacktestRunPlan{}
	enrichPlanFromSession(&plan, session)
	if plan.StockQuery != "小米" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.SignalQuery == "" {
		t.Fatal("expected signal from session")
	}
}

func TestFilterLegacyBacktestTools(t *testing.T) {
	schemas := []llm.ToolSchema{
		{Name: "run_strategy_backtest"},
		{Name: "generate_dca_strategy"},
		{Name: "loopback_strategy"},
	}
	out := FilterLegacyBacktestTools(schemas, "帮我回测一下", nil)
	if len(out) != 1 || out[0].Name != "run_strategy_backtest" {
		t.Fatalf("filtered=%v", out)
	}
}

func TestFilterSmartTradeBacktestTools(t *testing.T) {
	schemas := []llm.ToolSchema{
		{Name: "run_strategy_backtest"},
		{Name: "probe_bot_signal_series"},
	}
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	out := FilterSmartTradeBacktestTools(schemas, msg, nil)
	if len(out) != 1 || out[0].Name != "probe_bot_signal_series" {
		t.Fatalf("filtered=%v", out)
	}
	if len(FilterSmartTradeBacktestTools(schemas, "帮我回测一下", nil)) != 2 {
		t.Fatal("backtest intent should keep run_strategy_backtest")
	}
}

func TestRouteSignalProbeEvalMessage(t *testing.T) {
	skills := []string{"strategy-backtest-run", "strategy-backtest", "strategy-signal-probe"}
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	if p, ok := Route(skills, msg, nil); ok {
		t.Fatalf("signal probe eval should not route to playbook, got %q", p)
	}
}

func TestHeuristicBacktestPlanEvalStockCode(t *testing.T) {
	msg := "请帮我用「SAR抛物线」策略测试一下腾讯控股（0700.HK · 港股）。"
	plan := heuristicBacktestPlan(msg)
	if plan.StockQuery != "0700.HK" {
		t.Fatalf("stock=%q want 0700.HK", plan.StockQuery)
	}
	if plan.SignalQuery != "SAR抛物线" {
		t.Fatalf("signal=%q want SAR抛物线", plan.SignalQuery)
	}
}

func TestPickStockRowByCode(t *testing.T) {
	items := []map[string]any{
		{"code": "00700.HK", "name": "腾讯控股"},
		{"code": "01698.HK", "name": "腾讯音乐-SW"},
	}
	row, ok := pickStockRow(items, "0700.HK")
	if !ok || row["code"] != "00700.HK" {
		t.Fatalf("picked=%v ok=%v", row, ok)
	}
}

func TestPickCombination(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "a", "buy_signal": []any{map[string]any{"index": "SAR"}}, "frequency": []any{"5m", "60m", "daily"}},
		{"name": "RSI阈值信号", "signal_id": "b", "buy_signal": []any{map[string]any{"index": "RSI"}}},
	}
	row, err := pickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a" {
		t.Fatalf("picked=%v", row["signal_id"])
	}
	if got := tools.NormalizeCatalogFrequency(row["frequency"]); got != "60m" {
		t.Fatalf("frequency=%q want 60m", got)
	}
}

func TestPickCombinationDisambiguatesMultipleMatches(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "a"},
		{"name": "MACD金死叉配套SAR趋势", "signal_id": "b"},
		{"name": "SAR抛物线配套MACD与EMA复合策略", "signal_id": "c"},
	}
	row, err := pickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a" {
		t.Fatalf("picked=%v want a (SAR before MACD, shorter name)", row["signal_id"])
	}
}

func TestScoreCombinationMatchPrefersTokenOrder(t *testing.T) {
	tokens := signalTokens("SAR MACD")
	sarFirst := scoreCombinationMatch("SAR信号配套MACD直方图趋势", tokens)
	macdFirst := scoreCombinationMatch("MACD金死叉配套SAR趋势", tokens)
	if sarFirst <= macdFirst {
		t.Fatalf("sarFirst=%d macdFirst=%d", sarFirst, macdFirst)
	}
}
