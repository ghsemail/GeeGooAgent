package chat

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestResolveUsesCatalogRawWithoutSecondCatalogFetch(t *testing.T) {
	catalogCalls := 0
	flow := &Flow{
		StrategyQuery: "SAR",
		CatalogType:   catalogTypeCombination,
		CatalogLabel:  "SAR+MACD",
		CatalogRaw: map[string]any{
			"name": "SAR+MACD", "frequency": "60m",
			"buy_signal":  []any{map[string]any{"index": "SAR", "type": "signal"}},
			"sell_signal": []any{map[string]any{"index": "MACD", "type": "flag"}},
		},
	}
	r := &Runner{RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
		if req.Name == "get_signal_combinations" || req.Name == "get_index_signals" {
			catalogCalls++
		}
		return tools.Result{Status: tools.StatusError, Summary: "should not be called"}
	}}
	if err := r.ensureDiagnoseSignalResolved(context.Background(), nil, flow, tools.Context{}, nil, "read_strategy:resolve_signal"); err != nil {
		t.Fatal(err)
	}
	if catalogCalls != 0 {
		t.Fatalf("expected 0 catalog tool calls, got %d", catalogCalls)
	}
	if flow.DiagnoseSignalSource != "catalog_raw" || len(flow.ResolvedBuySignal) != 1 {
		t.Fatalf("source=%q buy=%d", flow.DiagnoseSignalSource, len(flow.ResolvedBuySignal))
	}
}

func TestSessionResolveCacheSharedAcrossLabToggle(t *testing.T) {
	session := &runtime.Session{}
	flowA := &Flow{
		StrategyQuery: "SAR", CatalogType: catalogTypeCombination, CatalogLabel: "SAR+MACD",
		CatalogRaw: map[string]any{
			"name": "SAR+MACD", "frequency": "60m",
			"buy_signal":  []any{map[string]any{"index": "SAR", "type": "signal"}},
			"sell_signal": []any{map[string]any{"index": "MACD", "type": "flag"}},
		},
	}
	r := &Runner{}
	if err := r.ensureDiagnoseSignalResolved(context.Background(), session, flowA, tools.Context{}, nil, "read_strategy:resolve_signal"); err != nil {
		t.Fatal(err)
	}
	flowB := &Flow{
		StrategyQuery: flowA.StrategyQuery, CatalogType: flowA.CatalogType, CatalogLabel: flowA.CatalogLabel,
		CatalogRaw: flowA.CatalogRaw, LabMode: true,
	}
	if err := r.ensureDiagnoseSignalResolved(context.Background(), session, flowB, tools.Context{}, nil, "read_strategy:resolve_signal"); err != nil {
		t.Fatal(err)
	}
	if flowA.ProbeSignalFingerprint != flowB.ProbeSignalFingerprint {
		t.Fatalf("fingerprint mismatch %q vs %q", flowA.ProbeSignalFingerprint, flowB.ProbeSignalFingerprint)
	}
}

func TestCardLabDefaultDetailRefWhenNoTriggers(t *testing.T) {
	flow := &Flow{Template: SkillSignalDiagnose, LabMode: true, LabEpisodeSummaries: strategydev.LabCardEpisodeSummaries{}}
	out := CardPayload(flow)
	if out["lab_default_detail_ref"] != strategydev.LabFlowDetailRef {
		t.Fatalf("ref=%v", out["lab_default_detail_ref"])
	}
	sums, _ := out["lab_episode_summaries"].(strategydev.LabCardEpisodeSummaries)
	if sums == nil || sums[strategydev.LabFlowDetailRef].DetailRef == "" {
		t.Fatal("expected flow:0 summary entry")
	}
}

func TestDiagnoseTimeNormalizedOnCard(t *testing.T) {
	raw := map[string]any{
		"verdict": "ok", "summary": "x", "range": map[string]any{"bar_count": 10},
		"buy_rules": []any{map[string]any{
			"index": "SAR",
			"last_bar": map[string]any{"time": "202610221130 00:00:00"},
			"sample_reasons": []any{map[string]any{"time": "202610221130 00:00:00", "reason": "hit"}},
		}},
	}
	d := diagnoseStructuredForCard(raw)
	rules := d["buy_rules"].([]any)
	row := rules[0].(map[string]any)
	lb := row["last_bar"].(map[string]any)
	if lb["time"] != "2026-10-22 11:30" {
		t.Fatalf("time=%v", lb["time"])
	}
}
