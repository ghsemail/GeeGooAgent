package strategydev

import (
	"fmt"
	"strings"
)

type tagSpec struct {
	longRegimes  []string
	shortRegimes []string
	side         string
	longOnly     bool
	shortOnly    bool
}

func nameBlob(c Candidate) string {
	return strings.ToUpper(strings.TrimSpace(c.Name))
}

func strategyTags(c Candidate) tagSpec {
	blob := nameBlob(c)
	if c.SignalID == specialRevertComboID {
		return tagSpec{longRegimes: []string{RegimeRevert}, side: SideLeft, longOnly: true}
	}
	if c.Kind == "combination" {
		return tagSpec{longRegimes: []string{RegimeUp, RegimeWeakUp}, side: SideRight, longOnly: true}
	}
	if c.Kind == "advanced" {
		return tagSpec{longRegimes: []string{RegimeUp, RegimeWeakUp}, side: SideRight, longOnly: true}
	}
	// index
	if strings.Contains(blob, "RSI") && strings.Contains(blob, "阈值") ||
		strings.Contains(blob, "KDJ") && strings.Contains(blob, "阈值") ||
		strings.Contains(blob, "布林带趋势") || strings.Contains(blob, "布林带突破") {
		return tagSpec{longRegimes: []string{RegimeRevert}, side: SideLeft, longOnly: true}
	}
	if strings.Contains(blob, "RSI") && (strings.Contains(blob, "金") || strings.Contains(blob, "死叉")) ||
		strings.Contains(blob, "MACD") && (strings.Contains(blob, "金") || strings.Contains(blob, "死叉")) ||
		strings.Contains(blob, "KDJ") {
		return tagSpec{
			longRegimes:  []string{RegimeUp, RegimeWeakUp},
			shortRegimes: []string{RegimeDown, RegimeWeakDown},
			side:         SideRight,
		}
	}
	// trend singles
	return tagSpec{
		longRegimes:  []string{RegimeUp, RegimeWeakUp},
		shortRegimes: []string{RegimeDown, RegimeWeakDown},
		side:         SideRight,
	}
}

func matchesRegimeSide(regime, side string, spec tagSpec, wantLong bool) bool {
	if spec.side != "" && side != SideNone && spec.side != side {
		return false
	}
	regs := spec.shortRegimes
	if wantLong {
		regs = spec.longRegimes
	}
	for _, r := range regs {
		if r == regime {
			return true
		}
	}
	return false
}

func advancedAllowed(c Candidate, freq string) bool {
	blob := nameBlob(c)
	f := NormalizeFrequency(freq)
	if strings.Contains(blob, "周期共振") || strings.Contains(blob, "MACD4") {
		return f == "5m"
	}
	if strings.Contains(blob, "4小时") || strings.Contains(blob, "4H") || strings.Contains(blob, "MACD4H") {
		return f == "60m"
	}
	if f == "weekly" || f == "1wk" {
		return false
	}
	return true
}

// FilterCandidates returns buy (combo+advanced) and sell (index) lists for the regime.
func FilterCandidates(cat Catalog, regime, side, freq string) (buy, sell []Candidate) {
	if !regimeAllowsSelection(regime) {
		return nil, nil
	}
	wantShort := regime == RegimeDown || regime == RegimeWeakDown
	wantLong := regime == RegimeUp || regime == RegimeWeakUp || regime == RegimeRevert

	for _, c := range cat.Indices {
		spec := strategyTags(c)
		if wantShort && matchesRegimeSide(regime, side, spec, false) && !spec.longOnly {
			sell = append(sell, c)
		}
	}
	for _, c := range cat.Combinations {
		spec := strategyTags(c)
		if wantLong && matchesRegimeSide(regime, side, spec, true) {
			if wantShort {
				continue
			}
			buy = append(buy, c)
		}
	}
	for _, c := range cat.Advanced {
		if !advancedAllowed(c, freq) {
			continue
		}
		spec := strategyTags(c)
		if wantLong && matchesRegimeSide(regime, side, spec, true) {
			if wantShort {
				continue
			}
			buy = append(buy, c)
		}
	}
	if wantShort {
		buy = coverBuyCandidates(cat, freq)
	}
	return buy, sell
}

func coverBuyCandidates(cat Catalog, freq string) []Candidate {
	if c := cat.findByID(coverComboID); c != nil && probeCompatible(c, freq) {
		return []Candidate{*c}
	}
	var out []Candidate
	for _, c := range cat.Indices {
		blob := nameBlob(c)
		if strings.Contains(blob, "吊灯信号") || strings.Contains(blob, "SAR信号") || strings.Contains(blob, "SAR 信号") {
			if probeCompatible(&c, freq) {
				out = append(out, c)
			}
		}
	}
	return out
}

func probeCompatible(c *Candidate, freq string) bool {
	if c == nil {
		return false
	}
	_ = freq
	return !isNoSignalRules(c.Buy)
}

func isNoSignalRules(rules []any) bool {
	if len(rules) == 0 {
		return true
	}
	for _, raw := range rules {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		idx := strings.ToLower(strings.TrimSpace(fmt.Sprint(row["index"])))
		if idx == "nosignal" || idx == "no_signal" {
			return true
		}
	}
	return false
}
