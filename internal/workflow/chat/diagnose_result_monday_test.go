package chat

import (
	"testing"

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
}
