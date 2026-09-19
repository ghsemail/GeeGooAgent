package premarket

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

func TestInferCapitalEvidenceAccumulation(t *testing.T) {
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {
				Code:           "00700.HK",
				CapitalMainIn:  1e8,
				CapitalFlowSummary: "**简要解读**：主力净流入",
			},
		},
	}
	args := buildKeyLevelsToolArgs(w, "00700.HK")
	cap, ok := args["capital_evidence"].(map[string]any)
	if !ok || cap["hint"] != "accumulation" {
		t.Fatalf("expected accumulation hint: %+v", args)
	}
}

func TestInferCapitalEvidenceSkipsDivergent(t *testing.T) {
	w := &memory.PreMarketWorking{
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {
				CapitalMainIn:              1e8,
				CapitalDistributionSummary: "**简要解读**：小单净流入而主力偏弱",
			},
		},
	}
	args := buildKeyLevelsToolArgs(w, "00700.HK")
	if _, ok := args["capital_evidence"]; ok {
		t.Fatalf("divergent capital should not send evidence: %+v", args)
	}
}
