package klinepattern

import "github.com/ghsemail/GeeGooAgent/internal/slots"

// clearNonAnchorBarPatternMarkers removes per-bar pattern hit lists used by FE chart markers.
// OHLC and window-level bias fields are kept for tables and summaries.
func clearNonAnchorBarPatternMarkers(ent *slots.KlinePatternBarWindowEntry) {
	if ent == nil || ent.Role == "anchor" {
		return
	}
	ent.KlinePatterns = nil
	ent.KlinePatternSummary = ""
	ent.KlinePatternBias = nil
}

// ThreeBarWindowForChartEmit returns a copy of win with pattern markers only on the anchor bar.
func ThreeBarWindowForChartEmit(win *slots.KlinePatternThreeBarWindow) *slots.KlinePatternThreeBarWindow {
	if win == nil {
		return nil
	}
	out := *win
	if len(win.Bars) == 0 {
		return &out
	}
	bars := make([]slots.KlinePatternBarWindowEntry, len(win.Bars))
	copy(bars, win.Bars)
	var windowHits []slots.KlinePatternHit
	for i := range bars {
		if bars[i].Omitted {
			continue
		}
		windowHits = append(windowHits, bars[i].KlinePatterns...)
	}
	for i := range bars {
		clearNonAnchorBarPatternMarkers(&bars[i])
	}
	// Lab 弹窗只读 anchor 列：形态若落在 prev/prev2，仍把摘要挂到 anchor 供 FE 展示名称。
	for i := range bars {
		if bars[i].Role != "anchor" || bars[i].Omitted {
			continue
		}
		if len(bars[i].KlinePatterns) == 0 && len(windowHits) > 0 {
			bars[i].KlinePatternSummary = slots.KlinePatternSummary(windowHits)
			b := slots.ScoreKlinePatternHits(windowHits)
			bars[i].KlinePatternBias = &b
		}
		break
	}
	out.Bars = bars
	return &out
}

// MTFContextForChartEmit strips non-anchor pattern markers from each timeframe three-bar window.
func MTFContextForChartEmit(ctx *slots.KlinePatternMTFContext) *slots.KlinePatternMTFContext {
	if ctx == nil {
		return nil
	}
	out := *ctx
	if len(ctx.Timeframes) == 0 {
		return &out
	}
	tfs := make([]slots.KlinePatternMTFEntry, len(ctx.Timeframes))
	copy(tfs, ctx.Timeframes)
	for i := range tfs {
		tfs[i].ThreeBar = ThreeBarWindowForChartEmit(tfs[i].ThreeBar)
	}
	out.Timeframes = tfs
	return &out
}
