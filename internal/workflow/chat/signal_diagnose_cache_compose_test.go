package chat

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestGlobalResolveCacheHit(t *testing.T) {
	key := "k-global-1"
	sig := slots.ResolvedSignal{
		Buy: []any{map[string]any{"index": "SAR", "type": "signal"}},
		Sell: []any{map[string]any{"index": "MACD", "type": "flag"}},
		Frequency: "60m", StrategyLabel: "combo",
	}
	storeSignalResolveGlobal(key, sig, "catalog_raw")
	ent, ok := loadSignalResolveGlobal(key)
	if !ok {
		t.Fatal("missing global entry")
	}
	flow := &Flow{}
	applySignalResolveCacheEntry(flow, ent)
	if !flow.DiagnoseSignalCached || flow.ResolvedStrategyLabel != "combo" {
		t.Fatalf("flow=%+v", flow)
	}
}

func TestLoadResolveCacheFromChatMetadataMapAny(t *testing.T) {
	ent := signalResolveCacheEntry{
		Buy: []any{map[string]any{"index": "SAR"}}, Sell: []any{map[string]any{"index": "MACD"}},
		Source: "catalog_raw", Frequency: "60m",
	}
	raw, _ := json.Marshal(ent)
	chat := &chatsession.ChatSession{Metadata: map[string]any{
		MetaKeySignalDiagnoseResolveCache: map[string]any{
			"abc123": json.RawMessage(raw),
		},
	}}
	rt := &runtime.Session{}
	loadSignalDiagnoseResolveCacheFromChat(chat, rt)
	if len(rt.SignalDiagnoseResolveCache) != 1 {
		t.Fatalf("cache=%v", rt.SignalDiagnoseResolveCache)
	}
}

func TestDiagnoseRangeTimesNormalized(t *testing.T) {
	raw := map[string]any{
		"verdict": "ok", "summary": "x", "range": map[string]any{
			"bar_count": 10,
			"start":     "202604211030 00:00:00",
			"end":       "202610221130 00:00:00",
		},
		"buy_rules": []any{map[string]any{"index": "SAR", "last_bar": map[string]any{"signal": 0}}},
	}
	d := diagnoseStructuredForCard(raw)
	rng := d["range"].(map[string]any)
	if rng["start"] != "2026-04-21 10:30" || rng["end"] != "2026-10-22 11:30" {
		t.Fatalf("range=%v", rng)
	}
}

func TestComposeSetsFallbackFields(t *testing.T) {
	flow := &Flow{
		Template: SkillSignalDiagnose, MonthsBack: 3,
		ProbeBuyHits: 1, ProbeSellHits: 1, ProbeBarCount: 10,
		Strategies: []StrategyItem{{Frequency: "60m"}},
	}
	r := &Runner{ComposeLLM: nil}
	text, err := r.synthesizeSignalDiagnoseJudgment(context.Background(), flow)
	if err != nil || text == "" {
		t.Fatal(err)
	}
	if !flow.ComposeFallback || flow.ComposeTimeoutMs != defaultSignalDiagnoseComposeDeadline.Milliseconds() {
		t.Fatalf("fallback=%v timeout=%d", flow.ComposeFallback, flow.ComposeTimeoutMs)
	}
	card := CardPayload(flow)
	if card["compose_fallback"] != true {
		t.Fatalf("card=%v", card["compose_fallback"])
	}
}

func TestResolveMCPUsesCatalogToolCache(t *testing.T) {
	calls := 0
	flow := &Flow{
		StrategyQuery: "SAR", CatalogType: catalogTypeCombination, CatalogLabel: "SAR Combo",
		CatalogRaw: map[string]any{"name": "incomplete"},
	}
	r := &Runner{RunTool: func(_ context.Context, req tools.CallRequest, _ tools.Context) tools.Result {
		if req.Name == "get_signal_combinations" {
			calls++
			return tools.Result{Status: tools.StatusOK, Data: map[string]any{
				"items": []any{map[string]any{
					"name": "SAR Combo", "frequency": "60m",
					"buy_signal":  []any{map[string]any{"index": "SAR", "type": "signal"}},
					"sell_signal": []any{map[string]any{"index": "MACD", "type": "flag"}},
				}},
			}}
		}
		return tools.Result{Status: tools.StatusError}
	}}
	_, _, _, err := r.resolveDiagnoseSignalUncached(context.Background(), flow, tools.Context{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _, _, err = r.resolveDiagnoseSignalUncached(context.Background(), flow, tools.Context{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 catalog fetch, got %d", calls)
	}
}

func TestSessionResolveCacheMetrics(t *testing.T) {
	session := &runtime.Session{}
	flow := &Flow{
		CatalogType: catalogTypeCombination, CatalogLabel: "SAR Combo",
		CatalogRaw: map[string]any{
			"name": "SAR Combo", "frequency": "60m",
			"buy_signal":  []any{map[string]any{"index": "SAR", "type": "signal"}},
			"sell_signal": []any{map[string]any{"index": "MACD", "type": "flag"}},
		},
	}
	r := &Runner{}
	if err := r.ensureDiagnoseSignalResolved(context.Background(), session, flow, tools.Context{}, nil, "read_strategy:resolve_signal"); err != nil {
		t.Fatal(err)
	}
	flow2 := &Flow{
		CatalogType: flow.CatalogType, CatalogLabel: flow.CatalogLabel, CatalogRaw: flow.CatalogRaw,
	}
	if err := r.ensureDiagnoseSignalResolved(context.Background(), session, flow2, tools.Context{}, nil, "read_strategy:resolve_signal"); err != nil {
		t.Fatal(err)
	}
	if flow2.PhaseTimingsMs["read_strategy:resolve_signal:cache_hit"] != 1 {
		t.Fatalf("timings=%v", flow2.PhaseTimingsMs)
	}
	if flow2.PhaseTimingsMs["read_strategy:resolve_signal:cache_scope"] != resolveCacheScopeSession {
		t.Fatalf("scope=%v", flow2.PhaseTimingsMs["read_strategy:resolve_signal:cache_scope"])
	}
}
