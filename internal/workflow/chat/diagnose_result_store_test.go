package chat

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestPersistSignalDiagnoseResultStepwiseLabModeSticky(t *testing.T) {
	rt := runtime.NewSession()
	lab := true
	rt.StickyPanelOps = &runtime.SignalDiagnoseOpts{LabMode: &lab}
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
	if !ok || ent == nil {
		t.Fatal("missing result")
	}
	if !ent.LabMode {
		t.Fatal("expected lab_mode on stored result")
	}
	if len(ent.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab_episode_summaries")
	}
	if ent.LabDefaultDetailRef != strategydev.LabFlowDetailRef {
		t.Fatalf("lab_default_detail_ref=%q", ent.LabDefaultDetailRef)
	}
	resp := MondayAPIResponse(ent)
	if resp.Lab == nil {
		t.Fatal("Monday response must include lab when lab_mode")
	}
}

func TestPersistSignalDiagnoseResultStepwiseLabModeInbound(t *testing.T) {
	rt := runtime.NewSession()
	lab := true
	rt.InboundPanelOps = &runtime.SignalDiagnoseOpts{LabMode: &lab}
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	bars := []any{map[string]any{"time": "t0", "close": 1.0}, map[string]any{"time": "t1", "close": 2.0}}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "bars": bars, "buy_merged": []any{1, 0}, "sell_merged": []any{0, 0},
	})
	ent, ok := LookupSignalDiagnoseResult(chat, rt.SignalDiagnoseResultLatest)
	if !ok || ent == nil || len(ent.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab summaries from inbound lab_mode")
	}
	if rt.InboundPanelOps == nil {
		t.Fatal("persist draft must not clear inbound panel ops")
	}
}

func TestPersistSignalDiagnoseResultStepwiseLabWithoutLabMode(t *testing.T) {
	rt := runtime.NewSession()
	off := false
	rt.StickyPanelOps = &runtime.SignalDiagnoseOpts{LabMode: &off}
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	bars := []any{map[string]any{"close": 1.0}, map[string]any{"close": 2.0}}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "bars": bars, "buy_merged": []any{1, 0}, "sell_merged": []any{0, 0},
	})
	ent, ok := LookupSignalDiagnoseResult(chat, rt.SignalDiagnoseResultLatest)
	if !ok || ent == nil {
		t.Fatal("missing result")
	}
	if ent.LabMode {
		t.Fatal("lab_mode should stay false on stored result")
	}
	if len(ent.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab_episode_summaries without lab_mode")
	}
	if ent.LabDefaultDetailRef != strategydev.LabFlowDetailRef {
		t.Fatalf("lab_default_detail_ref=%q", ent.LabDefaultDetailRef)
	}
	resp := MondayAPIResponse(ent)
	if resp.Lab == nil {
		t.Fatal("Monday response must include lab when probe has bars (no lab_mode)")
	}
}

func TestPersistSignalDiagnoseResultLongOnlyOmitsLabSell(t *testing.T) {
	rt := runtime.NewSession()
	td := runtime.TradeDirectionLongOnly
	rt.InboundPanelOps = &runtime.SignalDiagnoseOpts{TradeDirection: &td}
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	nBars := 24
	bars := make([]any, nBars)
	buyMerged := make([]any, nBars)
	sellMerged := make([]any, nBars)
	for i := range bars {
		bars[i] = map[string]any{"time": "t", "close": float64(i), "open": 1.0, "high": 1.0, "low": 1.0}
	}
	buyMerged[3] = 1
	for i := range sellMerged {
		if i%3 == 2 {
			sellMerged[i] = -1
		}
	}
	UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "000858.SZ", "frequency": "60m", "bars": bars,
		"buy_merged": buyMerged, "sell_merged": sellMerged,
	})
	ent, ok := LookupSignalDiagnoseResult(chat, rt.SignalDiagnoseResultLatest)
	if !ok || ent == nil {
		t.Fatal("missing result")
	}
	if len(ent.SignalEval.SellDetails) != 0 {
		t.Fatalf("signal_eval sell details=%d want 0", len(ent.SignalEval.SellDetails))
	}
	sellKeys := 0
	for ref := range ent.LabEpisodeSummaries {
		if strings.HasPrefix(ref, "sell:") {
			sellKeys++
		}
	}
	if sellKeys > 0 {
		t.Fatalf("lab must omit sell:* keys, got %d", sellKeys)
	}
	if resp := MondayAPIResponse(ent); resp.Lab != nil {
		if m, ok := resp.Lab.(strategydev.LabCardEpisodeSummaries); ok {
			for ref := range m {
				if strings.HasPrefix(ref, "sell:") {
					t.Fatalf("Monday lab still has %s", ref)
				}
			}
		}
	}
}

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
	if len(ent.LabEpisodeSummaries) == 0 {
		t.Fatal("expected lab summaries on probe persist without lab_mode opts")
	}
	if resp := MondayAPIResponse(ent); resp.Lab == nil {
		t.Fatal("Monday response must include lab after probe persist")
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
