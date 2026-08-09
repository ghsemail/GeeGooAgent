package stockfmt

import "testing"

func TestInferHourlyBiasFromTaggedConclusion(t *testing.T) {
	raw := "> **整体结论**：短线走势偏空，建议观望。"
	if got := InferHourlyBias("", raw, ""); got != "bearish" {
		t.Fatalf("expected bearish, got %q", got)
	}
}

func TestInferHourlyBiasNeutral(t *testing.T) {
	raw := "> **整体结论**：震荡整理，维持中性。"
	if got := InferHourlyBias(raw, "", ""); got != "" {
		t.Fatalf("expected neutral/empty, got %q", got)
	}
}

func TestHourlyContradictsBuy(t *testing.T) {
	signal := "MACD：0：偏空\nRSI：1：走弱"
	if !HourlyContradictsBuy("", signal, "") {
		t.Fatal("expected buy contradiction")
	}
}

func TestHourlyContradictsSell(t *testing.T) {
	price := "> **整体判断**：价格结构偏多，短线有望反弹。"
	if !HourlyContradictsSell(price, "", "") {
		t.Fatal("expected sell contradiction")
	}
}
