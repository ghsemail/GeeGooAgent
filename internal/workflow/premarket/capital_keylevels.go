package premarket

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// buildKeyLevelsToolArgs builds get_key_levels payload including capital hint (no fabricated price band).
func buildKeyLevelsToolArgs(w *memory.PreMarketWorking, code string) map[string]any {
	args := map[string]any{"code": code, "include_60m": true, "include_legacy": true}
	ws, ok := w.Stocks[code]
	if !ok {
		return args
	}
	if cap := inferCapitalEvidence(ws); cap != nil {
		args["capital_evidence"] = cap
	}
	return args
}

func inferCapitalEvidence(ws memory.StockWorkspace) map[string]any {
	capText := capitalInterpretationText(ws)
	if stockfmt.CapitalFlowDivergent(capText) {
		return nil
	}
	out := map[string]any{}
	if ws.CapitalMainIn > 0 {
		out["hint"] = "accumulation"
		out["label"] = strings.TrimSpace(capText)
		if out["label"] == "" {
			out["label"] = "主力净流入"
		}
		return out
	}
	if ws.CapitalMainIn < 0 {
		out["hint"] = "distribution"
		out["label"] = strings.TrimSpace(capText)
		if out["label"] == "" {
			out["label"] = "主力净流出"
		}
		return out
	}
	return nil
}
