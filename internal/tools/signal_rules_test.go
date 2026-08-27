package tools

import "testing"

func TestNormalizeSignalRulesFillsType(t *testing.T) {
	out := NormalizeSignalRules([]any{
		map[string]any{"index": "SAR"},
		map[string]any{"index": "MACD", "type": "flag"},
	})
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].(map[string]any)["type"] != "signal" {
		t.Fatalf("first type=%v", out[0].(map[string]any)["type"])
	}
	if out[1].(map[string]any)["type"] != "flag" {
		t.Fatalf("second type=%v", out[1].(map[string]any)["type"])
	}
}

func TestNormalizeCatalogFrequency(t *testing.T) {
	if got := NormalizeCatalogFrequency("daily"); got != "daily" {
		t.Fatalf("string=%q", got)
	}
	if got := NormalizeCatalogFrequency([]any{"5m", "60m", "daily"}); got != "60m" {
		t.Fatalf("array=%q want 60m", got)
	}
	if got := NormalizeCatalogFrequency([]string{"5m", "daily"}); got != "daily" {
		t.Fatalf("[]string=%q want daily", got)
	}
	if got := NormalizeCatalogFrequency(nil); got != "60m" {
		t.Fatalf("nil=%q", got)
	}
	if got := NormalizeCatalogFrequency("[5m 60m daily]"); got != "60m" {
		t.Fatalf("sprint-array=%q", got)
	}
}

func TestSanitizeBacktestSignalArgs(t *testing.T) {
	args := map[string]any{
		"buy_signal":  []any{map[string]any{"index": "SAR"}},
		"sell_signal": []any{map[string]any{"index": "MACD"}},
	}
	SanitizeBacktestSignalArgs(args)
	if args["sell_signal"].([]any)[0].(map[string]any)["type"] != "signal" {
		t.Fatal("expected sell type=signal")
	}
}

func TestNormalizeBacktestStrategyLinkFromSignalID(t *testing.T) {
	args := map[string]any{
		"signal_id": "662d0424c4cee7ffb800d0af",
	}
	NormalizeBacktestStrategyLink(args)
	ids, ok := args["strategy_ids"].([]string)
	if !ok || len(ids) != 1 || ids[0] != "662d0424c4cee7ffb800d0af" {
		t.Fatalf("strategy_ids=%v", args["strategy_ids"])
	}
}

func TestNormalizeBacktestStrategyLinkKeepsExistingIDs(t *testing.T) {
	args := map[string]any{
		"signal_id":     "aaa",
		"strategy_ids":  []any{"bbb"},
		"strategy_kind": "combination",
	}
	NormalizeBacktestStrategyLink(args)
	if args["strategy_ids"].([]any)[0] != "bbb" {
		t.Fatalf("should not overwrite existing ids: %v", args["strategy_ids"])
	}
}
