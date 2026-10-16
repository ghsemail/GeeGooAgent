package slots

import (
	"fmt"
	"strings"
)

// SARMACDCatalogSignalID is the live catalog combo「SAR信号配套MACD直方图趋势」
// (sell in DB remains nosignal; Agent overrides sell on probe/backtest/diagnose only).
const SARMACDCatalogSignalID = "662d0424c4cee7ffb800d0af"

// SARMACDStrategyBSellRule is the online default sell side for SAR+MACD combination
// (MACD death-cross via type=signal; catalog sell is typically nosignal).
func SARMACDStrategyBSellRule() map[string]any {
	return map[string]any{
		"index": "MACD",
		"type":  "signal",
		"param": map[string]any{
			"fastPeriod":   "12",
			"slowPeriod":   "26",
			"signalPeriod": "9",
		},
	}
}

func isAgentSARMACDCoverCombo(row map[string]any) bool {
	if row == nil {
		return false
	}
	if id := strings.TrimSpace(fmt.Sprint(row["signal_id"])); id == SARMACDCatalogSignalID {
		return true
	}
	name := strings.TrimSpace(fmt.Sprint(row["name"]))
	if name == "" {
		return false
	}
	upper := strings.ToUpper(name)
	// Canonical SAR+MACD histogram combo; skip broader mixes (e.g. SAR+MACD+RSI).
	return strings.Contains(name, "SAR信号配") && strings.Contains(upper, "MACD") &&
		strings.Contains(name, "直方图")
}

// IsAgentSARMACDCoverComboLabel reports whether a strategy display label is the cover combo.
func IsAgentSARMACDCoverComboLabel(label string) bool {
	return isAgentSARMACDCoverCombo(map[string]any{"name": strings.TrimSpace(label)})
}

func sellRulesAreMACDSignalOnly(sell []any) bool {
	if len(sell) != 1 {
		return false
	}
	row, ok := sell[0].(map[string]any)
	if !ok {
		return false
	}
	idx := strings.ToUpper(strings.TrimSpace(fmt.Sprint(row["index"])))
	typ := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["type"])))
	return idx == "MACD" && typ == "signal"
}

func applySARMACDStrategyBSellDefault(sig *ResolvedSignal, row map[string]any) {
	if sig == nil || !isAgentSARMACDCoverCombo(row) {
		return
	}
	if sellRulesAreMACDSignalOnly(sig.Sell) {
		return
	}
	sig.Sell = []any{SARMACDStrategyBSellRule()}
}

// SARMACDAgentRuleSummary is UI copy for the agent-side default buy/sell on SAR+MACD combos.
func SARMACDAgentRuleSummary(comboLabel string) string {
	if !IsAgentSARMACDCoverComboLabel(comboLabel) {
		return ""
	}
	return "买 SAR 信号 + MACD 直方图（flag）；卖 MACD 死叉（signal）"
}
