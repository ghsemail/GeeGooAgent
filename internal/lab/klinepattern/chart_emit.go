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
	for i := range bars {
		clearNonAnchorBarPatternMarkers(&bars[i])
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
