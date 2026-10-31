package chat

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestPersistSignalDiagnoseResultAfterProbeTool(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	bars := []any{map[string]any{"time": "t0", "close": 1.0}, map[string]any{"time": "t1", "close": 2.0}}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "frequency": "60m", "bars": bars,
		"buy_merged": []any{1, 0}, "sell_merged": []any{0, -1},
	})
	key := rt.SignalDiagnoseResultLatest
	if key == "" {
		t.Fatal("expected result_key")
	}
	ent, ok := LookupSignalDiagnoseResult(chat, key)
	if !ok || ent == nil {
		t.Fatal("result not in chat metadata")
	}
	if ent.ChartProbe["bars"] == nil {
		t.Fatal("chart_probe must include bars")
	}
	if ent.SignalEval.Method == "" {
		t.Fatal("expected computed signal_eval")
	}
	if ent.StockCode != "00700.HK" {
		t.Fatalf("stock=%q", ent.StockCode)
	}
}

func TestPersistSignalDiagnoseResultMergesDiagnose(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	bars := []any{map[string]any{"close": 1.0}}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "bars": bars, "buy_merged": []any{1}, "sell_merged": []any{0},
	})
	key := rt.SignalDiagnoseResultLatest
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "diagnose_bot_signal_series", map[string]any{
		"buy_rules": []any{map[string]any{"name": "rsi"}},
	})
	ent, ok := LookupSignalDiagnoseResult(chat, key)
	if !ok || ent == nil {
		t.Fatal("missing result")
	}
	if ent.DiagnoseDetail == nil {
		t.Fatal("expected diagnose_detail after diagnose tool")
	}
}

func TestFinalizeSignalDiagnoseResultFromFlowEvalJudgment(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	flow := &Flow{
		Template: SkillSignalDiagnose, RunID: "run-1", StockCode: "00700.HK",
		Status: StatusCompleted, Phase: PhaseDone, CardStage: "eval_done",
		EvalJudgment: "## 一、综合\n测试",
		ProbeRaw: map[string]any{
			"bars": []any{map[string]any{"close": 1.0}},
			"buy_merged": []any{1}, "sell_merged": []any{0},
		},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	SaveToSession(rt, flow)
	FinalizeSignalDiagnoseResultFromSession(rt, chat)
	ent, ok := LookupSignalDiagnoseResult(chat, "")
	if !ok || ent == nil {
		t.Fatal("expected latest result")
	}
	if ent.EvalJudgment == "" {
		t.Fatal("eval_judgment missing")
	}
	if ent.RunID != "run-1" {
		t.Fatalf("run_id=%q", ent.RunID)
	}
}
