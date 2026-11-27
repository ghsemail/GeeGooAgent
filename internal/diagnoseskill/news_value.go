package diagnoseskill

import "strings"

// News value rates whether an item is news and how much it matters to the pool. It
// is separate from the direction (neutral means no clear direction) and from
// analysis_status. All labels empty means unrated.

const (
	macroNewsValueVersion = "v1"
	macroNewsValueSource  = "model"

	validityNews    = "news"
	validityNonNews = "non_news"
	materialityNone = "none"
)

var (
	nonNewsTypes = map[string]struct{}{
		"quote_page": {}, "listing": {}, "data_table": {}, "promo": {}, "other": {},
	}
	materialities = map[string]struct{}{"high": {}, "medium": {}, "low": {}, materialityNone: {}}
)

// MacroNewsValue is the rating part of an item result.
type MacroNewsValue struct {
	Validity     string `json:"validity"`
	NonNewsType  string `json:"non_news_type"`
	Materiality  string `json:"materiality"`
	ValueReason  string `json:"value_reason"`
	ValueSource  string `json:"value_source"`
	ValueVersion string `json:"value_version"`
	// PoolEligible is computed here from the labels, never taken from the model.
	PoolEligible bool `json:"pool_eligible"`
}

func unratedValue() MacroNewsValue { return MacroNewsValue{PoolEligible: true} }

func label(raw any) string {
	s, _ := raw.(string)
	return strings.TrimSpace(s)
}

// resolveValue accepts a rating only as a whole. Anything missing or invalid leaves
// the item unrated; it never rejects the item or asks for the fallback model.
func resolveValue(rawValidity, rawType, rawMateriality, rawReason any) MacroNewsValue {
	out := MacroNewsValue{
		Validity:    strings.ToLower(label(rawValidity)),
		ValueReason: label(rawReason),
		ValueSource: macroNewsValueSource, ValueVersion: macroNewsValueVersion,
	}
	if !isMostlyHan(out.ValueReason) {
		return unratedValue()
	}
	out.ValueReason = clipRunes(out.ValueReason, 120)
	switch out.Validity {
	case validityNews:
		out.Materiality = strings.ToLower(label(rawMateriality))
		if _, ok := materialities[out.Materiality]; !ok {
			return unratedValue()
		}
	case validityNonNews:
		out.NonNewsType = strings.ToLower(label(rawType))
		if _, ok := nonNewsTypes[out.NonNewsType]; !ok {
			return unratedValue()
		}
	default:
		return unratedValue()
	}
	out.PoolEligible = out.Validity != validityNonNews && out.Materiality != materialityNone
	return out
}
