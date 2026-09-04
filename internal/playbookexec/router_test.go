package playbookexec

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
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
	if !procedural.BacktestRunIntent(msg) {
		t.Fatal("expected backtest intent")
	}
	if p, ok := Route(nil, msg, nil); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v want playbook without matched skills", p, ok)
	}
}

func TestRouteBacktestContinueWithSession(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 小米"})
	if p, ok := Route(nil, "就用刚才那套", session); !ok || p != playbookBacktestRun {
		t.Fatalf("route=%q ok=%v", p, ok)
	}
}

func TestRouteIgnoresParentSkillWithoutBacktestIntent(t *testing.T) {
	skills := []string{"strategy-backtest-run", "strategy-backtest"}
	for _, msg := range []string{
		"这个信号怎么样",
		"这个策略好不好",
		"刚才那个收益是多少",
		"你好",
	} {
		if p, ok := Route(skills, msg, nil); ok {
			t.Fatalf("casual %q routed to %q", msg, p)
		}
	}
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 小米"})
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: "## 小米 01810 · SmartTrade 回测\n\n- **策略**：SAR信号配套MACD直方图趋势"})
	if p, ok := Route(skills, "这个信号准不准", session); ok {
		t.Fatalf("follow-up question routed to %q", p)
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

func TestEnrichPlanReusesLastStrategyAndIgnoresSkillPrompt(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{
		Role: llm.RoleSystem,
		Content: "Relevant skill instructions (procedural memory). Follow only if applicable:\n" +
			"### strategy-backtest\n组合信号、共振、SAR、MACD、回测、测信号",
	})
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测一下小米"})
	session.AppendMessage(llm.Message{
		Role:    llm.RoleAssistant,
		Content: "## 小米 01810 · SmartTrade 回测\n\n- **策略**：SAR信号配套MACD直方图趋势\n- **收益率**：1.2%",
	})
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "这个信号准吗"})
	plan := BacktestRunPlan{}
	enrichPlanFromSession(&plan, session)
	if plan.SignalQuery != "SAR信号配套MACD直方图趋势" {
		t.Fatalf("signal=%q, want last chosen strategy, not skill prompt", plan.SignalQuery)
	}
	if strings.Contains(plan.SignalQuery, "Relevant skill") || strings.Contains(plan.SignalQuery, "测信号") {
		t.Fatalf("signal leaked from skill prompt: %q", plan.SignalQuery)
	}
}

func TestHeuristicDoesNotCopyWholeMessageAsSignal(t *testing.T) {
	plan := heuristicBacktestPlan("帮我用组合信号回测一下小米")
	if plan.StockQuery != "小米" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.SignalKind != "combination" {
		t.Fatalf("kind=%q", plan.SignalKind)
	}
	if plan.SignalQuery == "帮我用组合信号回测一下小米" {
		t.Fatal("signal_query should not be the entire user utterance")
	}
}

func TestExtractStockQueryChineseNameAndRejectsDaily(t *testing.T) {
	if got := slots.ExtractStockQuery("帮我回测一下中际旭创"); got != "中际旭创" {
		t.Fatalf("stock=%q want 中际旭创", got)
	}
	if got := slots.ExtractStockQuery("回测 300308"); got != "300308" {
		t.Fatalf("ashare=%q", got)
	}
	catalog := "当前共有 **6 个组合信号**（全部支持 5m /60m / daily三种频率）：\nSAR信号配套MACD直方图趋势"
	if got := slots.ExtractStockQuery(catalog); got != "" {
		t.Fatalf("catalog prose must not yield ticker, got %q", got)
	}
	if got := slots.ExtractStockQuery("帮我回测 MACD"); got != "" {
		t.Fatalf("indicator must not be stock, got %q", got)
	}
}

func TestZhongjiBacktestDoesNotInheritDailyFromCatalog(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "现在有哪些组合信号"})
	session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: "当前共有 **6 个组合信号**（全部支持 5m /60m / daily三种频率）：\n\n- **SAR信号配套MACD直方图趋势**"})
	plan := heuristicBacktestPlan("帮我回测一下中际旭创")
	enrichPlanFromSession(&plan, session)
	if plan.StockQuery != "中际旭创" {
		t.Fatalf("stock=%q", plan.StockQuery)
	}
	if plan.StockQuery == "DAILY" || strings.EqualFold(plan.StockQuery, "daily") {
		t.Fatal("daily frequency leaked as stock")
	}
	if strings.Contains(strings.ToUpper(plan.SignalQuery), "SAR") {
		t.Fatalf("catalog listing must not become selected signal, got %q", plan.SignalQuery)
	}
}

func TestCurrentStockOverridesSessionAndReusesConfirmedStrategy(t *testing.T) {
	session := runtime.NewSession()
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: "帮我回测 sar+macd 小米"})
	session.AppendMessage(llm.Message{
		Role:    llm.RoleAssistant,
		Content: "## 小米 01810 · SmartTrade 回测\n\n- **策略**：SAR信号配套MACD直方图趋势\n- **收益率**：1.2%",
	})
	plan := heuristicBacktestPlan("帮我回测一下中际旭创")
	enrichPlanFromSession(&plan, session)
	if plan.StockQuery != "中际旭创" {
		t.Fatalf("current stock must win, got %q", plan.StockQuery)
	}
	if plan.SignalQuery != "SAR信号配套MACD直方图趋势" {
		t.Fatalf("confirmed backtest strategy should reuse, got %q", plan.SignalQuery)
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

func TestPickCombination(t *testing.T) {
	items := []map[string]any{
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "a", "buy_signal": []any{map[string]any{"index": "SAR"}}},
		{"name": "RSI阈值信号", "signal_id": "b", "buy_signal": []any{map[string]any{"index": "RSI"}}},
	}
	row, err := slots.PickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatal(err)
	}
	if row["signal_id"] != "a" {
		t.Fatalf("picked=%v", row["signal_id"])
	}
}

func TestPickCombinationAutoPicksBestAmongMultiple(t *testing.T) {
	items := []map[string]any{
		{"name": "MACD金叉配SAR", "signal_id": "far"},
		{"name": "SAR信号配套MACD直方图趋势", "signal_id": "best"},
		{"name": "SAR+MACD+RSI 三指标组合", "signal_id": "extra"},
	}
	row, err := slots.PickCombination(items, "SAR MACD")
	if err != nil {
		t.Fatalf("multi-match should auto-pick, got %v", err)
	}
	name := fmt.Sprint(row["name"])
	if !strings.Contains(strings.ToUpper(name), "SAR") || !strings.Contains(strings.ToUpper(name), "MACD") {
		t.Fatalf("picked=%v name=%v", row["signal_id"], name)
	}
}
