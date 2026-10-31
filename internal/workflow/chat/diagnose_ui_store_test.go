package chat

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestCardPayloadOmitsBarsAndSignalEvalWithKey(t *testing.T) {
	bars := []any{
		map[string]any{"time": "t0", "close": 1.0},
		map[string]any{"time": "t1", "close": 2.0},
	}
	flow := &Flow{
		Template:   SkillSignalDiagnose,
		RunID:      "run-test",
		Status:     StatusCompleted,
		CardStage:  "eval_done",
		StockCode:  "00700.HK",
		ProbeRaw:   map[string]any{"bars": bars, "buy_merged": []any{0, 1}, "sell_merged": []any{0, -1}},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	out := CardPayloadForSession(rt, chat, flow)
	raw, _ := json.Marshal(out)
	if len(raw) == 0 {
		t.Fatal("empty card")
	}
	if out["diagnose_ui_key"] == nil || out["diagnose_ui_key"] == "" {
		t.Fatalf("expected diagnose_ui_key, got %v", out["diagnose_ui_key"])
	}
	if _, ok := out["signal_eval"]; ok {
		t.Fatal("signal_eval must not appear on SSE card")
	}
	cp, _ := out["chart_probe"].(map[string]any)
	if cp == nil {
		t.Fatal("expected light chart_probe")
	}
	if _, ok := cp["bars"]; ok {
		t.Fatal("chart_probe.bars must be omitted from SSE card")
	}
	if cp["bar_count"] != float64(2) && cp["bar_count"] != 2 {
		t.Fatalf("bar_count=%v", cp["bar_count"])
	}
	key, _ := out["diagnose_ui_key"].(string)
	ent, ok := LookupSignalDiagnoseUIPayload(chat, key)
	if !ok || ent == nil {
		t.Fatal("stored payload not found")
	}
	if ent.ChartProbe["bars"] == nil {
		t.Fatal("stored chart_probe must include bars")
	}
	if ent.SignalEval.Method == "" {
		t.Fatal("stored signal_eval missing")
	}
}

func TestPrepareSignalDiagnoseEmitCardUsesPendingProbe(t *testing.T) {
	rt := runtime.NewSession()
	bars := []any{map[string]any{"time": "t0", "close": 3.0}}
	RememberSignalDiagnoseProbe(rt, map[string]any{
		"code": "00700.HK", "frequency": "60m", "bars": bars,
		"buy_merged": []any{1}, "sell_merged": []any{0},
	})
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	card := map[string]any{
		"skill": "signal_diagnose", "phase": "done", "status": "completed",
		"stock_code": "00700.HK", "strategy_query": "SAR · 腾讯",
	}
	light := PrepareSignalDiagnoseEmitCard(rt, chat, card)
	if light["diagnose_ui_key"] == nil {
		t.Fatal("expected diagnose_ui_key")
	}
	if _, ok := light["signal_eval"]; ok {
		t.Fatal("signal_eval stripped from emit card")
	}
}
