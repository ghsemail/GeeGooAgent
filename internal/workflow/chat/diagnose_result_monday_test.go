package chat

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func TestMondayAPIResponseLabSummaries(t *testing.T) {
	ent := &SignalDiagnoseResult{
		ChartProbe: map[string]any{"bars": []any{}},
		LabEpisodeSummaries: strategydev.LabCardEpisodeSummaries{
			"buy:0": {DetailRef: "buy:0", StartIdx: 0},
		},
	}
	resp := MondayAPIResponse(ent)
	if resp.Lab == nil {
		t.Fatal("expected lab")
	}
	sums, ok := resp.Lab.(strategydev.LabCardEpisodeSummaries)
	if !ok || len(sums) != 1 {
		t.Fatalf("lab=%T %v", resp.Lab, resp.Lab)
	}
	if resp.TradeDirection != runtime.TradeDirectionBoth {
		t.Fatalf("trade_direction=%q want both", resp.TradeDirection)
	}
}

func TestMondayAPIResponseRunIDAndTradeDirection(t *testing.T) {
	ent := &SignalDiagnoseResult{
		RunID:          "flow-abc123",
		TradeDirection: runtime.TradeDirectionLongOnly,
		ChartProbe:     map[string]any{"bars": []any{}},
		SignalEval:     slots.SignalEpisodeEval{Method: "until_opposite_signal"},
	}
	resp := MondayAPIResponse(ent)
	if resp.RunID != "flow-abc123" {
		t.Fatalf("run_id=%q", resp.RunID)
	}
	if resp.TradeDirection != runtime.TradeDirectionLongOnly {
		t.Fatalf("trade_direction=%q", resp.TradeDirection)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["run_id"] != "flow-abc123" {
		t.Fatalf("json run_id=%v", m["run_id"])
	}
	if m["trade_direction"] != "long_only" {
		t.Fatalf("json trade_direction=%v", m["trade_direction"])
	}
}

func TestMondayAPIResponseTradeDirectionDefaultsBoth(t *testing.T) {
	ent := &SignalDiagnoseResult{
		ChartProbe: map[string]any{"bars": []any{}},
	}
	resp := MondayAPIResponse(ent)
	if resp.TradeDirection != runtime.TradeDirectionBoth {
		t.Fatalf("trade_direction=%q want both", resp.TradeDirection)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["trade_direction"] != "both" {
		t.Fatalf("json trade_direction=%v", m["trade_direction"])
	}
	if _, ok := m["run_id"]; ok {
		t.Fatalf("run_id should be omitted when empty, got %v", m["run_id"])
	}
}
