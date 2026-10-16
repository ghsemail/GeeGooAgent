package slots

import (
	"fmt"
	"testing"
)

func TestApplySARMACDStrategyBSellDefaultFromCatalogSAR(t *testing.T) {
	sig, err := rowToResolved(map[string]any{
		"name": "SAR信号配套MACD直方图趋势",
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		"sell_signal": []any{
			map[string]any{"index": "SAR", "param": map[string]any{}},
			map[string]any{"index": "MACD", "param": map[string]any{}},
		},
		"frequency": "60m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Sell) != 1 {
		t.Fatalf("sell=%v", sig.Sell)
	}
	row := sig.Sell[0].(map[string]any)
	if fmt.Sprint(row["index"]) != "MACD" || fmt.Sprint(row["type"]) != "signal" {
		t.Fatalf("sell rule=%v", row)
	}
	assertMACDSignalSellParam(t, row["param"])
}

func assertMACDSignalSellParam(t *testing.T, raw any) {
	t.Helper()
	p, ok := raw.(map[string]any)
	if !ok {
		t.Fatalf("param=%v", raw)
	}
	for k, want := range map[string]string{
		"fastPeriod": "12", "slowPeriod": "26", "signalPeriod": "9",
	} {
		if fmt.Sprint(p[k]) != want {
			t.Fatalf("param[%q]=%v want %q", k, p[k], want)
		}
	}
}

func TestSARMACDStrategyBSellRuleParamShape(t *testing.T) {
	row := SARMACDStrategyBSellRule()
	assertMACDSignalSellParam(t, row["param"])
}

func TestApplySARMACDStrategyBSellDefaultLeavesMACDSignal(t *testing.T) {
	want := SARMACDStrategyBSellRule()
	sig, err := rowToResolved(map[string]any{
		"name": "SAR信号配套MACD直方图趋势",
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		"sell_signal": []any{want},
		"frequency":   "60m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Sell) != 1 {
		t.Fatalf("sell=%v", sig.Sell)
	}
}

func TestApplySARMACDStrategyBSellDefaultFromCatalogNosignal(t *testing.T) {
	sig, err := rowToResolved(map[string]any{
		"name":      "SAR信号配套MACD直方图趋势",
		"signal_id": SARMACDCatalogSignalID,
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		"sell_signal": []any{
			map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}},
		},
		"frequency": "60m",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(sig.Sell) != 1 || fmt.Sprint(sig.Sell[0].(map[string]any)["index"]) != "MACD" {
		t.Fatalf("sell=%v", sig.Sell)
	}
}

func TestApplySARMACDStrategyBSellDefaultSkipsTripleCombo(t *testing.T) {
	sig, err := rowToResolved(map[string]any{
		"name": "SAR+MACD+RSI 三指标组合",
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		"sell_signal": []any{
			map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(sig.Sell[0].(map[string]any)["index"]) != "nosignal" {
		t.Fatalf("sell=%v", sig.Sell)
	}
}

func TestMergeProbeRuleOverridesSellWinsOverSARMACDDefault(t *testing.T) {
	base := ResolvedSignal{
		Buy: []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		Sell: []any{SARMACDStrategyBSellRule()},
	}
	sellOverride := []any{map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}}}
	out := MergeProbeRuleOverrides(base, nil, sellOverride)
	if fmt.Sprint(out.Sell[0].(map[string]any)["index"]) != "SAR" {
		t.Fatalf("sell=%v", out.Sell)
	}
}
