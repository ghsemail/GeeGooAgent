package chat

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestResolveFlowForLabDetail_PrefersPersistChartOverStaleSnapshot(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	end := time.Date(2026, 11, 5, 10, 30, 0, 0, loc)
	var lookback []any
	for d := end.AddDate(0, -5, 0); !d.After(end); d = d.Add(60 * time.Minute) {
		lookback = append(lookback, map[string]any{"time": d.Format("2006-01-02 15:04:05"), "close": 1.0})
	}
	staleProbe := map[string]any{
		"bars":          append([]any(nil), lookback...),
		"bars_lookback": append([]any(nil), lookback...),
		"buy_merged":    make([]any, len(lookback)),
		"sell_merged":   make([]any, len(lookback)),
	}
	clampedProbe := map[string]any{"bars": append([]any(nil), lookback...)}
	strategydev.ClampProbeSeriesToMonthsBack(clampedProbe, 3)
	clampedBars, _ := clampedProbe["bars"].([]any)
	clampedProbe["buy_merged"] = make([]any, len(clampedBars))
	clampedProbe["sell_merged"] = make([]any, len(clampedBars))

	runID := "flow-20261105T104344"
	rt := runtime.NewSession()
	chat := &chatsession.ChatSession{ID: rt.ID, Metadata: map[string]any{}}

	staleFlow := &Flow{
		RunID: runID, Template: SkillSignalDiagnose, StockCode: "00700.HK", MonthsBack: 3,
		ProbeRaw: staleProbe,
		Strategies: []StrategyItem{{Frequency: "60m"}},
	}
	UpsertLabDiagnoseSnapshot(chat, staleFlow)

	ent := SignalDiagnoseResult{
		RunID: runID, StockCode: "00700.HK", MonthsBack: 3, Frequency: "60m",
		ChartProbe: clampedProbe,
		SignalEval: slots.EvaluateSignalEpisodes(clampedBars, nil, nil),
		CreatedAt:  time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		ExpiresAt:  time.Now().UTC().Add(48 * time.Hour),
	}
	storeSignalDiagnoseResult(rt, chat, ent)

	got := ResolveFlowForLabDetail(chat, runID)
	if got == nil {
		t.Fatal("expected flow")
	}
	gotBars := strategydev.LabDetailChartBars(got.ProbeRaw)
	if len(gotBars) != len(clampedBars) {
		t.Fatalf("lab-detail flow bars len=%d want persisted chart %d (stale snapshot had %d)",
			len(gotBars), len(clampedBars), len(lookback))
	}
	first := gotBars[0].(map[string]any)["time"]
	wantFirst := clampedBars[0].(map[string]any)["time"]
	if first != wantFirst {
		t.Fatalf("first bar time=%v want %v", first, wantFirst)
	}
}
