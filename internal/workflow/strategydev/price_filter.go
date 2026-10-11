package strategydev

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// ApplyPriceAnalysisEpisodeFilter analyzes bars and masks disallowed trigger marks.
func ApplyPriceAnalysisEpisodeFilter(bars []any, freq string, buyMerged, sellMerged []any) ([]any, []any, PriceFilterResult, EpisodeTriggerFilterStats) {
	pf := AnalyzePriceFilter(bars, freq)
	buy, sell, stats := MaskEpisodeTriggers(bars, buyMerged, sellMerged, pf)
	return buy, sell, pf, stats
}

// PriceFilterResult is the pure-price (+volume) gate for strategy development.
type PriceFilterResult struct {
	Regime        string   `json:"regime"`
	Side          string   `json:"side"`
	AllowLong     bool     `json:"allow_long"`
	AllowShort    bool     `json:"allow_short"`
	VolumeSummary string   `json:"volume_summary,omitempty"`
	Notes         []string `json:"notes,omitempty"`
	RVOL          float64  `json:"rvol,omitempty"`
	ER            float64  `json:"er,omitempty"`
	BarCount      int      `json:"bar_count"`
	MinBars       int      `json:"min_bars"`
	Frequency     string   `json:"frequency,omitempty"`
}

func minBarsForPriceFilter(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 80
	case "60m", "15m":
		return 60
	case "daily":
		return 40
	case "weekly", "1wk":
		return 20
	default:
		return 60
	}
}

func swingKForFrequency(freq string) int {
	switch NormalizeFrequency(freq) {
	case "5m":
		return 5
	case "60m", "15m":
		return 3
	case "daily":
		return 2
	case "weekly", "1wk":
		return 1
	default:
		return 3
	}
}

type ohlcvBar struct {
	high, low, close, volume float64
}

func barsOHLCV(bars []any) []ohlcvBar {
	out := make([]ohlcvBar, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, ohlcvBar{
			high:   slots.FloatAny(row["high"]),
			low:    slots.FloatAny(row["low"]),
			close:  slots.FloatAny(row["close"]),
			volume: slots.FloatAny(row["volume"]),
		})
	}
	return out
}

// AnalyzePriceFilter classifies structure from OHLCV bars for one frequency.
func AnalyzePriceFilter(bars []any, freq string) PriceFilterResult {
	freq = NormalizeFrequency(freq)
	data := barsOHLCV(bars)
	minBars := minBarsForPriceFilter(freq)
	res := PriceFilterResult{
		BarCount:  len(data),
		MinBars:   minBars,
		Frequency: freq,
		Regime:    RegimeUnclear,
		Side:      SideNone,
	}
	if len(data) < minBars {
		res.Notes = append(res.Notes, fmt.Sprintf("K 线根数 %d 低于最低 %d，结构不明确", len(data), minBars))
		res.VolumeSummary = volumeSummary(data, 20)
		return res
	}

	k := swingKForFrequency(freq)
	sh, sl := collectSwingPoints(data, k)
	if len(sh) < 2 || len(sl) < 2 {
		res.Notes = append(res.Notes, "摆动点不足，结构不明确")
		res.VolumeSummary = volumeSummary(data, 20)
		res.ER = efficiencyRatio(data, 10)
		if res.ER > 0 {
			res.Notes = append(res.Notes, fmt.Sprintf("ER≈%.2f（附注）", res.ER))
		}
		return res
	}

	structRegime, structSide := structureFromSwings(sh, sl)
	res.Regime = structRegime
	res.Side = structSide

	last := len(data) - 1
	res.RVOL = relativeVolume(data, last, 20)
	res.ER = efficiencyRatio(data, 10)
	if res.ER > 0 {
		res.Notes = append(res.Notes, fmt.Sprintf("ER≈%.2f（附注）", res.ER))
	}

	bos, choch := detectStructureBreak(data, sh, sl, structRegime)
	if choch {
		res.Notes = append(res.Notes, "CHOCH：逆结构首次突破（仅预警）")
		if !bos {
			res.Regime = RegimeUnclear
			res.Side = SideNone
			res.Notes = append(res.Notes, "CHOCH 未获反向 BOS 或更大周期确认，结论不明确")
		}
	} else if bos {
		res.Notes = append(res.Notes, "BOS：顺结构突破（延续）")
		if res.RVOL >= 1 {
			res.Notes = append(res.Notes, fmt.Sprintf("突破根 RVOL=%.2f 放量确认", res.RVOL))
		} else if res.RVOL > 0 && res.RVOL < 1 {
			res.Notes = append(res.Notes, fmt.Sprintf("突破根 RVOL=%.2f 缩量可疑", res.RVOL))
			if structRegime == RegimeUp || structRegime == RegimeDown {
				res.Regime = RegimeUnclear
				res.Side = SideNone
			}
		}
	}

	halfNote := structureWindowVolumeNote(data, min(len(data), minBars))
	if halfNote != "" {
		res.Notes = append(res.Notes, halfNote)
	}
	res.VolumeSummary = volumeSummary(data, 20)

	res.AllowLong, res.AllowShort = allowDirectionsFromFilter(res)
	return res
}

func structureFromSwings(sh, sl []swingPoint) (regime, side string) {
	h1, h2 := sh[len(sh)-2], sh[len(sh)-1]
	l1, l2 := sl[len(sl)-2], sl[len(sl)-1]
	hh, hl := h2.price > h1.price, l2.price > l1.price
	lh, ll := h2.price < h1.price, l2.price < l1.price
	switch {
	case hh && hl:
		return RegimeUp, SideRight
	case lh && ll:
		return RegimeDown, SideRight
	default:
		return RegimeRange, SideNone
	}
}

type swingPoint struct {
	idx   int
	price float64
}

func collectSwingPoints(data []ohlcvBar, k int) (highs, lows []swingPoint) {
	if k < 1 {
		k = 1
	}
	for i := k; i < len(data)-k; i++ {
		if isSwingHigh(data, i, k) {
			highs = append(highs, swingPoint{idx: i, price: data[i].high})
		}
		if isSwingLow(data, i, k) {
			lows = append(lows, swingPoint{idx: i, price: data[i].low})
		}
	}
	return highs, lows
}

func isSwingHigh(data []ohlcvBar, i, k int) bool {
	h := data[i].high
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if data[j].high > h {
			return false
		}
	}
	return true
}

func isSwingLow(data []ohlcvBar, i, k int) bool {
	l := data[i].low
	for j := i - k; j <= i+k; j++ {
		if j == i {
			continue
		}
		if data[j].low < l {
			return false
		}
	}
	return true
}

func detectStructureBreak(data []ohlcvBar, sh, sl []swingPoint, structRegime string) (bos, choch bool) {
	if len(data) == 0 || len(sh) == 0 || len(sl) == 0 {
		return false, false
	}
	last := data[len(data)-1]
	lastSH := sh[len(sh)-1]
	lastSL := sl[len(sl)-1]
	switch structRegime {
	case RegimeUp:
		if last.close > lastSH.price || last.high > lastSH.price {
			return true, false
		}
		if last.close < lastSL.price || last.low < lastSL.price {
			return false, true
		}
	case RegimeDown:
		if last.close < lastSL.price || last.low < lastSL.price {
			return true, false
		}
		if last.close > lastSH.price || last.high > lastSH.price {
			return false, true
		}
	}
	return false, false
}

func relativeVolume(data []ohlcvBar, idx, window int) float64 {
	if idx < 0 || idx >= len(data) || window <= 0 {
		return 0
	}
	start := idx - window
	if start < 0 {
		start = 0
	}
	var sum float64
	var n int
	for i := start; i < idx; i++ {
		if data[i].volume > 0 {
			sum += data[i].volume
			n++
		}
	}
	if n == 0 || data[idx].volume <= 0 {
		return 0
	}
	return data[idx].volume / (sum / float64(n))
}

func efficiencyRatio(data []ohlcvBar, period int) float64 {
	if period <= 0 || len(data) <= period {
		return 0
	}
	n := len(data) - 1
	start := n - period
	if start < 0 {
		start = 0
	}
	change := math.Abs(data[n].close - data[start].close)
	var path float64
	for i := start + 1; i <= n; i++ {
		path += math.Abs(data[i].close - data[i-1].close)
	}
	if path <= 0 {
		return 0
	}
	return change / path
}

func volumeSummary(data []ohlcvBar, window int) string {
	if len(data) == 0 {
		return "无量能数据"
	}
	idx := len(data) - 1
	rvol := relativeVolume(data, idx, window)
	if rvol <= 0 {
		return "近均量对比不足"
	}
	switch {
	case rvol >= 1.5:
		return fmt.Sprintf("当根 RVOL=%.2f，相对近 %d 均量明显放量", rvol, window)
	case rvol >= 1:
		return fmt.Sprintf("当根 RVOL=%.2f，相对近 %d 均量温和放量", rvol, window)
	default:
		return fmt.Sprintf("当根 RVOL=%.2f，相对近 %d 均量缩量", rvol, window)
	}
}

func structureWindowVolumeNote(data []ohlcvBar, window int) string {
	if window < 4 || len(data) < window {
		return ""
	}
	seg := data[len(data)-window:]
	mid := len(seg) / 2
	var v1, v2 float64
	for i, b := range seg {
		if i < mid {
			v1 += b.volume
		} else {
			v2 += b.volume
		}
	}
	if v1 <= 0 && v2 <= 0 {
		return ""
	}
	p1 := seg[0].close
	p2 := seg[len(seg)-1].close
	volUp := v2 > v1*1.05
	volDown := v2 < v1*0.95
	switch {
	case p2 > p1 && volUp:
		return "结构窗：价升量增（同向附注）"
	case p2 > p1 && volDown:
		return "结构窗：价升量缩（背离附注）"
	case p2 < p1 && volUp:
		return "结构窗：价跌量增（同向附注）"
	case p2 < p1 && volDown:
		return "结构窗：价跌量缩（背离附注）"
	default:
		return "结构窗：量价关系中性（附注）"
	}
}

func allowDirectionsFromFilter(pf PriceFilterResult) (allowLong, allowShort bool) {
	suspicious := strings.Contains(strings.Join(pf.Notes, " "), "缩量可疑") ||
		(pf.RVOL > 0 && pf.RVOL < 1 && strings.Contains(strings.Join(pf.Notes, " "), "CHOCH"))
	longOK := pf.Regime == RegimeUp && !suspicious
	shortOK := pf.Regime == RegimeDown && !suspicious
	if strings.Contains(strings.Join(pf.Notes, " "), "CHOCH") && pf.Regime == RegimeUnclear {
		longOK, shortOK = false, false
	}
	return longOK, shortOK
}

// UsePriceAnalysisEpisodeFilterEnabled returns true when nil (default on).
func UsePriceAnalysisEpisodeFilterEnabled(enabled *bool) bool {
	if enabled == nil {
		return true
	}
	return *enabled
}

// StrippedEpisodeTrigger is one buy/sell trigger removed by the price gate (for diagnose reporting).
type StrippedEpisodeTrigger struct {
	Index int     `json:"index,omitempty"`
	Time  string  `json:"time"`
	Close float64 `json:"close"`
}

// EpisodeTriggerFilterStats counts triggers removed by the price gate and lists stripped bars.
type EpisodeTriggerFilterStats struct {
	PreFilterBuyHits  int `json:"pre_filter_buy_hits"`
	PreFilterSellHits int `json:"pre_filter_sell_hits"`
	PostFilterBuyHits int `json:"post_filter_buy_hits"`
	PostFilterSellHits int `json:"post_filter_sell_hits"`
	StrippedBuy  int `json:"stripped_buy"`
	StrippedSell int `json:"stripped_sell"`
	PostFilterBuyEpisodes         int `json:"post_filter_buy_episodes,omitempty"`
	PostFilterSellEpisodes        int `json:"post_filter_sell_episodes,omitempty"`
	PostFilterBuyEpisodesComplete  int `json:"post_filter_buy_episodes_complete,omitempty"`
	PostFilterSellEpisodesComplete int `json:"post_filter_sell_episodes_complete,omitempty"`
	StrippedBuyTriggers  []StrippedEpisodeTrigger `json:"stripped_buy_triggers,omitempty"`
	StrippedSellTriggers []StrippedEpisodeTrigger `json:"stripped_sell_triggers,omitempty"`
	InterceptionBasis    string                   `json:"interception_basis,omitempty"`
}

// EnrichEpisodeFilterStatsPostEval fills episode counts after masked episode evaluation.
func EnrichEpisodeFilterStatsPostEval(stats *EpisodeTriggerFilterStats, eval slots.SignalEpisodeEval) {
	if stats == nil {
		return
	}
	stats.PostFilterBuyEpisodes = len(eval.BuyDetails)
	stats.PostFilterSellEpisodes = len(eval.SellDetails)
	stats.PostFilterBuyEpisodesComplete = eval.BuyEpisodes.CompleteCount
	stats.PostFilterSellEpisodesComplete = eval.SellEpisodes.CompleteCount
}

// MaskEpisodeTriggers zeros buy/sell merged marks disallowed by the price filter.
func MaskEpisodeTriggers(bars []any, buyMerged, sellMerged []any, pf PriceFilterResult) ([]any, []any, EpisodeTriggerFilterStats) {
	stats := EpisodeTriggerFilterStats{
		InterceptionBasis: BuildPriceFilterInterceptionBasis(pf),
	}
	if len(buyMerged) == 0 && len(sellMerged) == 0 {
		return buyMerged, sellMerged, stats
	}
	stats.PreFilterBuyHits = countMergedSignal(buyMerged, 1)
	stats.PreFilterSellHits = countMergedSignal(sellMerged, -1)
	n := len(buyMerged)
	if len(sellMerged) > n {
		n = len(sellMerged)
	}
	outBuy := make([]any, n)
	outSell := make([]any, n)
	for i := 0; i < n; i++ {
		bv := 0
		if i < len(buyMerged) {
			bv = int(slots.FloatAny(buyMerged[i]))
		}
		sv := 0
		if i < len(sellMerged) {
			sv = int(slots.FloatAny(sellMerged[i]))
		}
		if bv == 1 && !pf.AllowLong {
			stats.StrippedBuy++
			stats.StrippedBuyTriggers = append(stats.StrippedBuyTriggers, triggerLedgerEntry(bars, i))
			bv = 0
		}
		if sv == -1 && !pf.AllowShort {
			stats.StrippedSell++
			stats.StrippedSellTriggers = append(stats.StrippedSellTriggers, triggerLedgerEntry(bars, i))
			sv = 0
		}
		outBuy[i] = bv
		outSell[i] = sv
	}
	stats.PostFilterBuyHits = countMergedSignal(outBuy, 1)
	stats.PostFilterSellHits = countMergedSignal(outSell, -1)
	return outBuy, outSell, stats
}

func countMergedSignal(merged []any, target int) int {
	n := 0
	for _, raw := range merged {
		if int(slots.FloatAny(raw)) == target {
			n++
		}
	}
	return n
}

func triggerLedgerEntry(bars []any, i int) StrippedEpisodeTrigger {
	ent := StrippedEpisodeTrigger{Index: i}
	if i >= 0 && i < len(bars) {
		if row, ok := bars[i].(map[string]any); ok {
			ent.Time = formatBarTimeAsiaShanghai(row)
			ent.Close = slots.FloatAny(row["close"])
		}
	}
	if ent.Time == "" {
		ent.Time = fmt.Sprintf("bar[%d]", i)
	}
	return ent
}

func formatBarTimeAsiaShanghai(row map[string]any) string {
	ts := barTimeUnix(row)
	if ts.IsZero() {
		return strings.TrimSpace(fmt.Sprint(row["time"]))
	}
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return ts.Format("2006-01-02 15:04")
	}
	return ts.In(loc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
}

func FormatPriceFilterNote(pf PriceFilterResult) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s · %s", pf.Regime, pf.Side)
	if pf.VolumeSummary != "" {
		fmt.Fprintf(&b, " · %s", pf.VolumeSummary)
	}
	if len(pf.Notes) > 0 {
		fmt.Fprintf(&b, " · %s", strings.Join(pf.Notes, "；"))
	}
	return b.String()
}
