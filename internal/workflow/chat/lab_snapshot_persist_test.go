package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestResolveFlowForLabDetailFromPersistedResultOnly(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	bars := []any{
		map[string]any{"time": "t0", "close": 1.0, "open": 1.0, "high": 1.0, "low": 1.0},
		map[string]any{"time": "t1", "close": 2.0, "open": 2.0, "high": 2.0, "low": 2.0},
	}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "frequency": "60m", "bars": bars,
		"buy_merged": []any{1, 0}, "sell_merged": []any{0, -1},
	})
	ent, ok := LookupSignalDiagnoseResult(chat, rt.SignalDiagnoseResultLatest)
	if !ok || ent == nil || ent.RunID == "" {
		t.Fatalf("expected persisted result with run_id ent=%+v", ent)
	}
	if _, ok := chat.Metadata[MetaKeyLabDiagnoseSnapshots]; !ok {
		t.Fatal("expected lab snapshot written on persist")
	}
	mondayRun := "6ae5e1ca7e27c6bcad2e4384"
	got := ResolveFlowForLabDetail(chat, mondayRun)
	if got == nil {
		t.Fatal("expected flow rebuilt from persist result when snapshot key differs")
	}
	if got.RunID != mondayRun {
		t.Fatalf("run_id=%q want Monday client id for cache key", got.RunID)
	}
	if len(got.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab summaries on rebuilt flow")
	}
	ref := strategydev.LabDetailRef("buy", 0)
	if _, ok := got.LabEpisodeSummaries[ref]; !ok {
		t.Fatalf("missing ref %s in %v", ref, got.LabEpisodeSummaries)
	}
}

func TestLookupSignalDiagnoseResultMondayClientRunIDFallback(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "bars": []any{map[string]any{"close": 1.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
	})
	ent, ok := LookupSignalDiagnoseResultForMonday(chat, "monday-run-doc-id")
	if !ok || ent == nil {
		t.Fatal("expected fallback to sole persisted result")
	}
	if ent.RunID == "monday-run-doc-id" {
		t.Fatal("stored run_id should remain agent flow id")
	}
}
