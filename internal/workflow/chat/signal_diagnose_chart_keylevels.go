package chat

import "strings"

func keyLevelSnapshotToMap(kl KeyLevelSnapshot) map[string]any {
	if kl.SupportLow <= 0 && kl.ResistHigh <= 0 && kl.SupportHigh <= 0 {
		return nil
	}
	return map[string]any{
		"support_low":    kl.SupportLow,
		"support_high":   kl.SupportHigh,
		"support_center": kl.SupportCenter,
		"resist_low":     kl.ResistLow,
		"resist_high":    kl.ResistHigh,
		"resist_center":  kl.ResistCenter,
		"summary":        kl.Summary,
	}
}

func keyLevelBarSeriesToMap(s *KeyLevelBarSeries) map[string]any {
	if s == nil || len(s.SupportLow) == 0 && len(s.ResistHigh) == 0 {
		return nil
	}
	out := map[string]any{"align": s.Align}
	putSlice(out, "support_low", s.SupportLow)
	putSlice(out, "support_high", s.SupportHigh)
	putSlice(out, "support_center", s.SupportCenter)
	putSlice(out, "resist_low", s.ResistLow)
	putSlice(out, "resist_high", s.ResistHigh)
	putSlice(out, "resist_center", s.ResistCenter)
	return out
}

func putSlice(m map[string]any, key string, v []float64) {
	if len(v) == 0 {
		return
	}
	m[key] = v
}

func keyBreakRefPrice(flow *Flow, epEndIdx int, strictEndReason string) float64 {
	if !strings.HasPrefix(strictEndReason, "key_break_") {
		return 0
	}
	field := strings.TrimPrefix(strictEndReason, "key_break_")
	if flow != nil && flow.KeyLevelSeries != nil {
		if p := seriesFloatAt(flow.KeyLevelSeries, epEndIdx, field); p > 0 {
			return p
		}
	}
	if flow == nil {
		return 0
	}
	kl := flow.KeyLevels
	switch field {
	case KeyBreakRefSupportLow:
		return kl.SupportLow
	case KeyBreakRefSupportHigh:
		return kl.SupportHigh
	case KeyBreakRefSupportCenter:
		return kl.SupportCenter
	case KeyBreakRefResistLow:
		return kl.ResistLow
	case KeyBreakRefResistHigh:
		return kl.ResistHigh
	case KeyBreakRefResistCenter:
		return kl.ResistCenter
	}
	return 0
}

func seriesFloatAt(s *KeyLevelBarSeries, idx int, field string) float64 {
	if s == nil || idx < 0 {
		return 0
	}
	var src []float64
	switch field {
	case KeyBreakRefSupportLow:
		src = s.SupportLow
	case KeyBreakRefSupportHigh:
		src = s.SupportHigh
	case KeyBreakRefSupportCenter:
		src = s.SupportCenter
	case KeyBreakRefResistLow:
		src = s.ResistLow
	case KeyBreakRefResistHigh:
		src = s.ResistHigh
	case KeyBreakRefResistCenter:
		src = s.ResistCenter
	default:
		return 0
	}
	if idx >= len(src) {
		return 0
	}
	return src[idx]
}
