package cognition

import "testing"

func TestPlanAfterPresetClarifyAmbiguousStockChoice(t *testing.T) {
	parent := planForDomain(DomainAmbiguous)
	parent.Mode = ModeClarify
	got := PlanAfterPresetClarify(parent, "个股/指标分析")
	if got.Domain != DomainStockAnalysis || got.Mode != ModeGather {
		t.Fatalf("got %s/%s", got.Domain, got.Mode)
	}
	if got.ClarifyQuestion != "" {
		t.Fatalf("clarify fields should be cleared: %+v", got)
	}
}

func TestPlanAfterPresetClarifyDCAGather(t *testing.T) {
	parent := planForDomain(DomainDCAGrid)
	parent.Mode = ModeClarify
	got := PlanAfterPresetClarify(parent, "组合信号（DCA 用）：多指标共振")
	if got.Domain != DomainDCAGrid || got.Mode != ModeGather {
		t.Fatalf("got %s/%s", got.Domain, got.Mode)
	}
}

func TestMapClarifyChoiceBacktestPreview(t *testing.T) {
	msg := "让我先看历史回测结果再决定（我会用动态止盈 + 固定止损的推荐组合先跑）"
	d, _, ok := mapClarifyChoice(msg)
	if !ok || d != DomainBacktestRun {
		t.Fatalf("mapClarifyChoice(%q) = %q ok=%v", msg, d, ok)
	}
}

func TestExtractJSONObjectCodeFence(t *testing.T) {
	raw, ok := extractJSONObject("Here you go:\n```json\n{\"domain\":\"chat\",\"mode\":\"talk\"}\n```")
	if !ok {
		t.Fatal("expected json")
	}
	if raw != `{"domain":"chat","mode":"talk"}` {
		t.Fatalf("raw=%q", raw)
	}
}
