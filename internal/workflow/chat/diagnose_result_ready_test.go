package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestSignalDiagnoseResultReadyEnvelope(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	flow := &Flow{
		Template: SkillSignalDiagnose, RunID: "flow-env-1", StockCode: "600519.SH",
		Status: StatusCompleted,
		ProbeRaw: map[string]any{
			"bars": []any{map[string]any{"close": 1.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
		},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	SaveToSession(rt, flow)
	key := FinalizeSignalDiagnoseResultFromSession(rt, chat)
	if key == "" {
		t.Fatal("expected result key from finalize")
	}
	env := SignalDiagnoseResultReadyEnvelope(chat, key)
	if env == nil {
		t.Fatal("nil envelope")
	}
	if env["phase"] != "final" {
		t.Fatalf("phase=%v", env["phase"])
	}
	if env["run_id"] != "flow-env-1" {
		t.Fatalf("run_id=%v", env["run_id"])
	}
	if env["stock_code"] != "600519.SH" {
		t.Fatalf("stock_code=%v", env["stock_code"])
	}
	if _, ok := env["chart_probe"]; ok {
		t.Fatal("envelope must not include chart_probe")
	}
}

func TestFinalizeReturnsEmptyWithoutProbe(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	if key := FinalizeSignalDiagnoseResultFromSession(rt, chat); key != "" {
		t.Fatalf("key=%q want empty", key)
	}
}
