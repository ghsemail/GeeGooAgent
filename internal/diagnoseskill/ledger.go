package diagnoseskill

import (
	"strings"
)

// MacroSymbolLedgerRow is the canonical macro daily symbol result shape (assemble-only; no HTTP PUT).
type MacroSymbolLedgerRow struct {
	Tier            string  `json:"tier"`
	Date            string  `json:"date"`
	Symbol          string  `json:"symbol"`
	PoolWorkflow    string  `json:"pool_workflow,omitempty"`
	Status          string  `json:"status"`
	JEVDirection    string  `json:"jev_direction"`
	JEVReason       string  `json:"jev_reason"`
	Confidence      float64 `json:"confidence,omitempty"`
	DetailRef       string  `json:"detail_ref,omitempty"`
	AgentSessionID  string  `json:"agent_session_id,omitempty"`
	AgentRunID      string  `json:"agent_run_id,omitempty"`
}

// DirectionToLedger maps diagnose JEV direction (bull/bear/unclear) to macro-api ledger enums.
func DirectionToLedger(direction string) string {
	switch strings.TrimSpace(direction) {
	case "bull":
		return "long"
	case "bear":
		return "short"
	default:
		return "neutral"
	}
}

// AssembleMacroSymbolLedgerRow builds a ledger row from a macro diagnose skill result (no write).
func AssembleMacroSymbolLedgerRow(
	date, symbol, sessionID, runID, poolWorkflow string,
	unified UnifiedJEV,
) MacroSymbolLedgerRow {
	st := "success"
	if !unified.OK {
		st = "failed"
	}
	return MacroSymbolLedgerRow{
		Tier: "symbol", Date: strings.TrimSpace(date), Symbol: strings.TrimSpace(symbol),
		PoolWorkflow: strings.TrimSpace(poolWorkflow), Status: st,
		JEVDirection: DirectionToLedger(unified.Direction),
		JEVReason:    unified.RecommendedReason,
		Confidence:   unified.Confidence,
		DetailRef:    "diagnose_macro",
		AgentSessionID: strings.TrimSpace(sessionID),
		AgentRunID:     strings.TrimSpace(runID),
	}
}