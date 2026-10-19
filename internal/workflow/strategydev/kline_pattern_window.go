package strategydev

import (
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// BuildKlinePatternThreeBarWindow builds prev2/prev/anchor OHLC + patterns around anchorIdx (trigger bar).
func BuildKlinePatternThreeBarWindow(bars []any, patternHits []any, anchorIdx int) *slots.KlinePatternThreeBarWindow {
	if !KlinePatternDisplayEnabled() || anchorIdx < 0 || anchorIdx >= len(bars) {
		return nil
	}
	win := &slots.KlinePatternThreeBarWindow{AnchorIndex: anchorIdx}
	var notes []string
	for _, spec := range []struct {
		role string
		idx  int
		note string
	}{
		{"prev2", anchorIdx - 2, "无前前一根 K 线"},
		{"prev", anchorIdx - 1, "无前一根 K 线"},
		{"anchor", anchorIdx, ""},
	} {
		ent := slots.KlinePatternBarWindowEntry{Role: spec.role}
		if spec.idx < 0 {
			ent.Omitted = true
			ent.OmitReason = "before_series_start"
			if spec.note != "" {
				notes = append(notes, spec.note)
			}
			win.Bars = append(win.Bars, ent)
			continue
		}
		if spec.idx >= len(bars) {
			ent.Omitted = true
			ent.OmitReason = "after_series_end"
			win.Bars = append(win.Bars, ent)
			continue
		}
		row, ok := bars[spec.idx].(map[string]any)
		if !ok {
			ent.Omitted = true
			ent.OmitReason = "invalid_bar"
			win.Bars = append(win.Bars, ent)
			continue
		}
		ent.Index = spec.idx
		ent.Time = formatBarTimeAsiaShanghai(row)
		ent.Open = slots.FloatAny(row["open"])
		ent.High = slots.FloatAny(row["high"])
		ent.Low = slots.FloatAny(row["low"])
		ent.Close = slots.FloatAny(row["close"])
		hits := FilteredKlinePatternsAtIndex(patternHits, spec.idx)
		if len(hits) > 0 {
			ent.KlinePatterns = hits
			ent.KlinePatternSummary = slots.KlinePatternSummary(hits)
			b := slots.ScoreKlinePatternHits(hits)
			ent.KlinePatternBias = &b
		}
		win.Bars = append(win.Bars, ent)
	}
	if len(notes) > 0 {
		win.WindowNote = strings.Join(notes, "；")
	}
	if anchorHits := FilteredKlinePatternsAtIndex(patternHits, anchorIdx); len(anchorHits) > 0 {
		b := slots.ScoreKlinePatternHits(anchorHits)
		win.Bias = &b
	}
	var windowHits []slots.KlinePatternHit
	for _, b := range win.Bars {
		if b.Omitted {
			continue
		}
		windowHits = append(windowHits, b.KlinePatterns...)
	}
	if len(windowHits) > 0 {
		wb := slots.ScoreKlinePatternHits(windowHits)
		win.WindowBias = &wb
	}
	return win
}

func alignBarIndexAtOrBefore(bars []any, trigger time.Time) int {
	if trigger.IsZero() || len(bars) == 0 {
		return -1
	}
	bestIdx := -1
	var bestTs time.Time
	for i, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		ts := barTimeUnix(row)
		if ts.IsZero() || ts.After(trigger) {
			continue
		}
		if bestIdx < 0 || ts.After(bestTs) {
			bestIdx = i
			bestTs = ts
		}
	}
	return bestIdx
}

func triggerTimeFromBars(bars []any, idx int) time.Time {
	if idx < 0 || idx >= len(bars) {
		return time.Time{}
	}
	row, ok := bars[idx].(map[string]any)
	if !ok {
		return time.Time{}
	}
	return barTimeUnix(row)
}

func mtfAlignNote(freq string, alignedIdx int, fetchOK bool) string {
	if !fetchOK {
		return fmt.Sprintf("%s 拉数失败（仅展示信号周期）", timeframeDisplayLabel(freq))
	}
	if alignedIdx < 0 {
		return fmt.Sprintf("%s 未找到对齐 bar", timeframeDisplayLabel(freq))
	}
	return ""
}
