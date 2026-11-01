package agent

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	workflowchat "github.com/ghsemail/GeeGooAgent/internal/workflow/chat"
)

func TestSyncChatFromRuntimeSetsSignalDiagnoseResultReadySSE(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	flow := &workflowchat.Flow{
		Template: workflowchat.SkillSignalDiagnose, RunID: "flow-abc", StockCode: "00700.HK",
		Status: workflowchat.StatusCompleted,
		ProbeRaw: map[string]any{
			"bars": []any{map[string]any{"close": 1.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
		},
		SignalEval: slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	workflowchat.SaveToSession(rt, flow)

	SyncChatFromRuntime(chat, rt, nil)

	if rt.SignalDiagnoseResultReadySSE == nil {
		t.Fatal("expected ready SSE payload after finalize")
	}
	if rt.SignalDiagnoseResultReadySSE["session_id"] != chat.ID {
		t.Fatalf("session_id=%v", rt.SignalDiagnoseResultReadySSE["session_id"])
	}
	if rt.SignalDiagnoseResultReadySSE["run_id"] != "flow-abc" {
		t.Fatalf("run_id=%v", rt.SignalDiagnoseResultReadySSE["run_id"])
	}
	if rt.SignalDiagnoseResultReadySSE["phase"] != "final" {
		t.Fatalf("phase=%v", rt.SignalDiagnoseResultReadySSE["phase"])
	}
	if _, ok := rt.SignalDiagnoseResultReadySSE["result_key"].(string); !ok {
		t.Fatalf("result_key=%v", rt.SignalDiagnoseResultReadySSE["result_key"])
	}
}

func TestMidTurnPersistDoesNotSetSignalDiagnoseResultReadySSE(t *testing.T) {
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}
	workflowchat.UpdateSignalDiagnoseResultAfterTool(rt, chat, "probe_bot_signal_series", map[string]any{
		"code": "00700.HK", "bars": []any{map[string]any{"close": 1.0}}, "buy_merged": []any{1}, "sell_merged": []any{0},
	})
	if rt.SignalDiagnoseResultReadySSE != nil {
		t.Fatalf("mid-turn persist must not set ready SSE: %+v", rt.SignalDiagnoseResultReadySSE)
	}
}
