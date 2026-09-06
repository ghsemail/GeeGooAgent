package slots

import (
	"fmt"
	"strings"
	"testing"
)

func TestNormalizeSignalRulesPairInheritsBuyType(t *testing.T) {
	buy := []any{map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}}}
	sell := []any{map[string]any{"index": "MACD", "param": map[string]any{}}}
	_, outSell := normalizeSignalRulesPair(buy, sell)
	if len(outSell) != 1 {
		t.Fatalf("sell=%v", outSell)
	}
	row := outSell[0].(map[string]any)
	if fmt.Sprint(row["type"]) != "flag" {
		t.Fatalf("type=%v row=%#v", row["type"], row)
	}
}

func TestNormalizeSignalChainFillsMissingSellType(t *testing.T) {
	sig, err := rowToResolved(map[string]any{
		"name":      "SAR信号配套MACD直方图趋势",
		"frequency": "60m",
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal", "param": map[string]any{}},
			map[string]any{"index": "MACD", "type": "flag", "param": map[string]any{}},
		},
		"sell_signal": []any{
			map[string]any{"index": "SAR", "param": map[string]any{}},
			map[string]any{"index": "MACD", "param": map[string]any{}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sell0 := sig.Sell[0].(map[string]any)
	if strings.TrimSpace(fmt.Sprint(sell0["type"])) == "" {
		t.Fatalf("sell[0] missing type: %v", sig.Sell)
	}
	sell1Resolved := sig.Sell[1].(map[string]any)
	if fmt.Sprint(sell1Resolved["type"]) != "flag" {
		t.Fatalf("sell[1].type=%v want flag", sell1Resolved["type"])
	}
}

func TestNormalizeSignalChainNosignal(t *testing.T) {
	out := NormalizeSignalChain([]any{
		map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}},
	})
	row := out[0].(map[string]any)
	if strings.TrimSpace(fmt.Sprint(row["type"])) == "" {
		t.Fatalf("nosignal type still empty: %v", row)
	}
}

func TestNormalizeSignalChainValidationShape(t *testing.T) {
	sig, err := rowToResolved(map[string]any{
		"name": "test",
		"buy_signal": []any{
			map[string]any{"index": "SAR", "type": "signal"},
		},
		"sell_signal": []any{
			map[string]any{"index": "SAR"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i, chain := range [][]any{sig.Buy, sig.Sell} {
		for j, raw := range chain {
			row := raw.(map[string]any)
			if strings.TrimSpace(fmt.Sprint(row["index"])) == "" {
				t.Fatalf("chain %d[%d] missing index", i, j)
			}
			if strings.TrimSpace(fmt.Sprint(row["type"])) == "" {
				t.Fatalf("chain %d[%d] missing type", i, j)
			}
			if row["param"] == nil {
				t.Fatalf("chain %d[%d] missing param", i, j)
			}
		}
	}
}
