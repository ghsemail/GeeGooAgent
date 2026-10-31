package runtime

import "strings"

// TradeDirection controls whether signal_diagnose evaluates short (sell) episodes.
type TradeDirection string

const (
	TradeDirectionBoth     TradeDirection = "both"
	TradeDirectionLongOnly TradeDirection = "long_only"
)

// ParseTradeDirection normalizes skill_options.signal_diagnose.trade_direction (legacy workflow_options).
// Unknown values default to both (same as omitted).
func ParseTradeDirection(raw string) TradeDirection {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "long_only":
		return TradeDirectionLongOnly
	default:
		return TradeDirectionBoth
	}
}

// LongOnlyEpisodeEval reports whether sell-side episodes should be skipped.
func (d TradeDirection) LongOnlyEpisodeEval() bool {
	return strings.EqualFold(strings.TrimSpace(string(d)), string(TradeDirectionLongOnly))
}
