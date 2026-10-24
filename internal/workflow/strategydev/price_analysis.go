package strategydev

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// PriceAnalysis is a multi-timeframe pure-price table for diagnose / workflow cards.
type PriceAnalysis struct {
	Code                string              `json:"code,omitempty"`
	SpotPrice           float64             `json:"spot_price,omitempty"`
	SpotLabel           string              `json:"spot_label,omitempty"`
	Rows                []PriceAnalysisRow  `json:"rows"`
	SupportResistance   string              `json:"support_resistance,omitempty"`
	Summary             string              `json:"summary,omitempty"`
	FilterTimeframe     string              `json:"filter_timeframe,omitempty"`
	FilterTimeframeNote string              `json:"filter_timeframe_note,omitempty"`
	AnchorTime          string              `json:"anchor_time,omitempty"`
	AnchorClose         float64             `json:"anchor_close,omitempty"`
	PriceScore          float64             `json:"price_score"`
	TFScores            map[string]TFScoreBreakdown `json:"tf_scores,omitempty"`
	ScoreReason         string              `json:"score_reason,omitempty"`
	PriceAnalysisJEV    *PriceAnalysisJEV   `json:"price_analysis_jev,omitempty"`
}

// PriceAnalysisRow is one row (周期) in the multi-TF table.
type PriceAnalysisRow struct {
	Timeframe  string  `json:"timeframe"`
	Conclusion string  `json:"conclusion"`
	Structure  string  `json:"structure"`
	BosChoch   string  `json:"bos_choch"`
	Position   string  `json:"position"`
	ER         float64 `json:"er,omitempty"`
	RVOL       float64 `json:"rvol,omitempty"`
	VolumeNote string  `json:"volume_note"`
}

// KeyLevelBands optional support/resistance for the footer line.
type KeyLevelBands struct {
	SupportLow  float64
	SupportHigh float64
	ResistLow   float64
	ResistHigh  float64
}

var multiTimeframeSpecs = []struct {
	Label string
	Freq  string
}{
	{"5m", "5m"},
	{"60m", "60m"},
	{"4h", "4h"},
	{"日线", "daily"},
	{"周线", "weekly"},
}

var nosignalProbe = []any{
	map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}},
}

const (
	structureFetchFailed              = "拉数失败"
	structureNoBarsBeforeAnchor       = "锚点前无 K 线"
	structureInsufficientBars         = "根数不足"
	structurePriorTradingDayUnavailable = "前一交易日锚点不可用"
)

// BuildRunPriceAnalysis fetches OHLCV for standard TFs and builds the table (latest window from now).
func BuildRunPriceAnalysis(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	spotPrice float64,
	kl KeyLevelBands,
	filterFreq string,
) PriceAnalysis {
	return buildRunPriceAnalysis(ctx, run, code, months, nil, spotPrice, kl, filterFreq)
}

// BuildRunPriceAnalysisAsOf is the single-point entry: four TFs probed with as_of/end at asOf (requires Signal as_of).
func BuildRunPriceAnalysisAsOf(
	ctx context.Context,
	run ToolRunner,
	code string,
	asOf time.Time,
	months int,
	spotPrice float64,
	kl KeyLevelBands,
	filterFreq string,
) PriceAnalysis {
	if asOf.IsZero() {
		return buildRunPriceAnalysis(ctx, run, code, months, nil, spotPrice, kl, filterFreq)
	}
	end := asOf
	return buildRunPriceAnalysis(ctx, run, code, months, &end, spotPrice, kl, filterFreq)
}

func buildRunPriceAnalysis(
	ctx context.Context,
	run ToolRunner,
	code string,
	months int,
	asOf *time.Time,
	spotPrice float64,
	kl KeyLevelBands,
	filterFreq string,
) PriceAnalysis {
	if !PriceAnalysisFeatureEnabled() {
		return PriceAnalysis{}
	}
	out := PriceAnalysis{
		Code:            strings.TrimSpace(code),
		SpotPrice:       spotPrice,
		FilterTimeframe: timeframeDisplayLabel(filterFreq),
	}
	if out.SpotPrice > 0 {
		out.SpotLabel = fmt.Sprintf("现价 %.2f", out.SpotPrice)
	}
	if asOf != nil && !asOf.IsZero() {
		loc, _ := time.LoadLocation("Asia/Shanghai")
		out.AnchorTime = asOf.In(loc).Format("2006-01-02 15:04") + " (Asia/Shanghai)"
	}
	if run != nil && strings.TrimSpace(code) != "" {
		months = CapMonthsBack(months)
		barsByFreq, fetchOKByFreq := FetchMultiTimeframePriceAnalysisBars(ctx, MultiTimeframePriceBarsInput{
			Run: run, Code: code, Months: months, AsOf: asOf, GateFreq: filterFreq,
		})
		for _, spec := range multiTimeframeSpecsForGate(filterFreq) {
			bars := barsByFreq[spec.Freq]
			fetchOK := fetchOKByFreq[spec.Freq]
			if !fetchOK || len(bars) == 0 {
				reason := structureFetchFailed
				if fetchOK && len(bars) == 0 {
					reason = structureInsufficientBars
				}
				out.Rows = append(out.Rows, emptyPriceAnalysisRow(spec.Label, reason))
				continue
			}
			if asOf != nil && !asOf.IsZero() {
				bars = truncateBarsAtAnchor(bars, *asOf)
			}
			if len(bars) == 0 {
				srcLen := len(barsByFreq[spec.Freq])
				out.Rows = append(out.Rows, emptyPriceAnalysisRow(spec.Label, structureReasonAfterAnchorTruncate(srcLen, fetchOK)))
				continue
			}
			analysisBars := tailBarsForPriceAnalysis(bars, spec.Freq)
			out.Rows = append(out.Rows, BuildPriceAnalysisRow(analysisBars, spec.Label, spec.Freq))
		}
	}
	if out.SpotPrice <= 0 && len(out.Rows) > 0 {
		// Prefer daily row bars if we fetched any.
		for _, spec := range multiTimeframeSpecs {
			if spec.Freq != "daily" {
				continue
			}
			if run == nil {
				break
			}
			bars, _ := fetchPriceAnalysisBarsWithAsOf(ctx, run, code, spec.Freq, months, nil)
			if sp := lastClose(bars); sp > 0 {
				out.SpotPrice = sp
				out.SpotLabel = fmt.Sprintf("现价 %.2g", sp)
			}
			break
		}
	}
	out.SupportResistance = formatSupportResistanceLine(kl, out.Rows)
	out.Summary = SummarizeMultiTimeframeRows(out.Rows)
	if ff := NormalizeFrequency(filterFreq); ff != "" {
		out.FilterTimeframeNote = fmt.Sprintf("Episode 方向门控使用 %s 周期结论", timeframeDisplayLabel(ff))
	}
	return out
}

// BuildPriceAnalysisFromBars builds a table when bars are already available (tests / offline).
// gateFreq selects upward TFs only; empty gateFreq keeps the full standard four-TF table.
func BuildPriceAnalysisFromBars(barsByFreq map[string][]any, spot float64, kl KeyLevelBands, gateFreq string) PriceAnalysis {
	out := PriceAnalysis{SpotPrice: spot}
	if spot > 0 {
		out.SpotLabel = fmt.Sprintf("现价 %.2f", spot)
	}
	for _, spec := range multiTimeframeSpecsForGate(gateFreq) {
		bars := barsByFreq[spec.Freq]
		if len(bars) == 0 {
			out.Rows = append(out.Rows, PriceAnalysisRow{
				Timeframe: spec.Label, Conclusion: RegimeUnclear, Structure: "K 线不足",
				BosChoch: "无", Position: "-", VolumeNote: "-",
			})
			continue
		}
		out.Rows = append(out.Rows, BuildPriceAnalysisRow(bars, spec.Label, spec.Freq))
	}
	out.SupportResistance = formatSupportResistanceLine(kl, out.Rows)
	out.Summary = SummarizeMultiTimeframeRows(out.Rows)
	return out
}

// BuildPriceAnalysisRow analyzes one timeframe window.
func BuildPriceAnalysisRow(bars []any, tfLabel, freq string) PriceAnalysisRow {
	row := PriceAnalysisRow{Timeframe: tfLabel, BosChoch: "无", Position: "-", VolumeNote: "-"}
	pf := AnalyzePriceFilter(bars, freq)
	row.Conclusion = pf.Regime
	row.ER = pf.ER
	row.RVOL = pf.RVOL

	data := barsOHLCV(bars)
	minBars := minBarsForPriceFilter(NormalizeFrequency(freq))
	if len(data) < minBars {
		row.Structure = structureInsufficientBars
		if len(pf.Notes) > 0 {
			row.VolumeNote = strings.TrimPrefix(pf.Notes[0], "K 线根数")
		}
		return row
	}
	k := swingKForFrequency(NormalizeFrequency(freq))
	sh, sl := collectSwingPoints(data, k)
	row.Structure = formatSwingStructureText(sh, sl)
	if len(data) >= minBars && len(sh) >= 2 && len(sl) >= 2 {
		structRegime, _ := structureFromSwings(sh, sl)
		bos, choch := detectStructureBreak(data, sh, sl, structRegime)
		row.BosChoch = formatBosChochNote(bos, choch, pf.RVOL)
		row.Position = rangePositionLabel(data, sh, sl)
		row.VolumeNote = compactVolumeNote(data, min(len(data), minBars), structRegime)
	} else if len(pf.Notes) > 0 {
		row.VolumeNote = strings.TrimPrefix(pf.Notes[0], "K 线根数")
	}
	return row
}

const (
	priceAnalysisProbeMaxAttempts = 3
	priceAnalysisProbeRetryBase   = 200 * time.Millisecond
)

func fetchPriceAnalysisBarsWithAsOf(ctx context.Context, run ToolRunner, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	if !PriceAnalysisFeatureEnabled() || run == nil {
		return nil, false
	}
	key := probeCacheKey(code, freq, months, 0, asOf)
	if bars, ok, hit := lookupPriceProbeCache(ctx, key); hit {
		return bars, ok
	}
	var lastSummary string
	for attempt := 1; attempt <= priceAnalysisProbeMaxAttempts; attempt++ {
		res := run(ctx, "probe_bot_signal_series", PriceAnalysisProbeArgsFrom(PriceAnalysisProbeParams{
			Code: code, Freq: freq, Months: months, AsOf: asOf, ExtendedFetch: true,
		}))
		if res.Status == tools.StatusOK {
			bars, _ := res.Data["bars"].([]any)
			storePriceProbeCache(ctx, key, bars, true)
			return bars, true
		}
		lastSummary = res.Summary
		if attempt < priceAnalysisProbeMaxAttempts && ctx.Err() == nil {
			delay := priceAnalysisProbeRetryBase * time.Duration(attempt)
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				storePriceProbeCache(ctx, key, nil, false)
				return nil, false
			case <-timer.C:
			}
		}
	}
	_ = lastSummary
	storePriceProbeCache(ctx, key, nil, false)
	return nil, false
}

func emptyPriceAnalysisRow(tfLabel, structure string) PriceAnalysisRow {
	return PriceAnalysisRow{
		Timeframe:  tfLabel,
		Conclusion: RegimeUnclear,
		Structure:  structure,
		BosChoch:   "无",
		Position:   "-",
		VolumeNote: "-",
	}
}

func structureReasonAfterAnchorTruncate(srcLen int, fetchOK bool) string {
	if !fetchOK || srcLen == 0 {
		return structureFetchFailed
	}
	return structureNoBarsBeforeAnchor
}

func lastClose(bars []any) float64 {
	if len(bars) == 0 {
		return 0
	}
	if row, ok := bars[len(bars)-1].(map[string]any); ok {
		return slots.FloatAny(row["close"])
	}
	return 0
}

func formatSwingStructureText(sh, sl []swingPoint) string {
	if len(sh) < 2 || len(sl) < 2 {
		return "摆动点不足"
	}
	h1, h2 := sh[len(sh)-2], sh[len(sh)-1]
	l1, l2 := sl[len(sl)-2], sl[len(sl)-1]
	tag := swingStructureTag(h1.price, h2.price, l1.price, l2.price)
	return fmt.Sprintf("高 %.2g→%.2g, 低 %.2g→%.2g (%s)", h1.price, h2.price, l1.price, l2.price, tag)
}

func swingStructureTag(h1, h2, l1, l2 float64) string {
	hh, hl := h2 > h1, l2 > l1
	lh, ll := h2 < h1, l2 < l1
	switch {
	case hh && hl:
		return "HH+HL"
	case lh && ll:
		return "LH+LL"
	case hh && ll:
		return "HH+LL"
	case lh && hl:
		return "LH+HL"
	default:
		return "混合"
	}
}

func formatBosChochNote(bos, choch bool, rvol float64) string {
	switch {
	case choch && bos:
		return "CHOCH+BOS"
	case choch:
		return "CHOCH（预警）"
	case bos && rvol >= 1:
		return "BOS 放量"
	case bos:
		return "BOS"
	default:
		return "无"
	}
}

func rangePositionLabel(data []ohlcvBar, sh, sl []swingPoint) string {
	if len(data) == 0 {
		return "-"
	}
	lo, hi := rangeBounds(data, sh, sl)
	if hi <= lo {
		return "中部"
	}
	close := data[len(data)-1].close
	ratio := (close - lo) / (hi - lo)
	switch {
	case ratio >= 0.72:
		return "靠近上沿"
	case ratio <= 0.28:
		return "靠近下沿"
	default:
		return "中部"
	}
}

func rangeBounds(data []ohlcvBar, sh, sl []swingPoint) (lo, hi float64) {
	if len(sh) > 0 && len(sl) > 0 {
		lo = sl[len(sl)-1].price
		hi = sh[len(sh)-1].price
		for _, p := range sl {
			if p.price < lo {
				lo = p.price
			}
		}
		for _, p := range sh {
			if p.price > hi {
				hi = p.price
			}
		}
		return lo, hi
	}
	lo, hi = data[0].low, data[0].high
	for _, b := range data {
		if b.low < lo {
			lo = b.low
		}
		if b.high > hi {
			hi = b.high
		}
	}
	return lo, hi
}

func compactVolumeNote(data []ohlcvBar, window int, structRegime string) string {
	if window < 4 || len(data) < window {
		return "-"
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
	p1, p2 := seg[0].close, seg[len(seg)-1].close
	dir := "横盘"
	switch {
	case p2 > p1*1.002:
		dir = "上涨"
	case p2 < p1*0.998:
		dir = "下跌"
	}
	volPhrase := "后半中性"
	if v1 > 0 {
		switch {
		case v2 > v1*1.05:
			volPhrase = "后半放量"
		case v2 < v1*0.95:
			volPhrase = "后半缩量"
		}
	}
	rel := "中性"
	switch {
	case dir == "上涨" && volPhrase == "后半放量":
		rel = "同向"
	case dir == "下跌" && volPhrase == "后半放量":
		rel = "同向"
	case dir == "上涨" && volPhrase == "后半缩量":
		rel = "背离"
	case dir == "下跌" && volPhrase == "后半缩量":
		rel = "背离"
	}
	if dir == "横盘" {
		return "横盘+" + volPhrase
	}
	_ = structRegime
	return fmt.Sprintf("%s+%s, %s", dir, volPhrase, rel)
}

func formatSupportResistanceLine(kl KeyLevelBands, rows []PriceAnalysisRow) string {
	if kl.SupportLow > 0 && kl.ResistHigh > 0 {
		sHi := kl.SupportHigh
		if sHi <= kl.SupportLow {
			sHi = kl.SupportLow
		}
		rLo := kl.ResistLow
		if rLo <= 0 || rLo > kl.ResistHigh {
			rLo = kl.ResistHigh
		}
		return fmt.Sprintf("支撑 %.2f-%.2f, 阻力 %.2f-%.2f", kl.SupportLow, sHi, rLo, kl.ResistHigh)
	}
	// Fallback: daily-ish range from rows if present.
	for _, r := range rows {
		if r.Timeframe != "日线" {
			continue
		}
		if strings.Contains(r.Structure, "低 ") {
			return inferSRFromStructure(r.Structure)
		}
	}
	return ""
}

func inferSRFromStructure(structure string) string {
	// Best-effort only; key levels preferred.
	return ""
}

func timeframeDisplayLabel(freq string) string {
	switch NormalizeFrequency(freq) {
	case "5m":
		return "5m"
	case "15m":
		return "15m"
	case "30m":
		return "30m"
	case "60m":
		return "60m"
	case "4h":
		return "4h"
	case "daily":
		return "日线"
	case "weekly", "1wk":
		return "周线"
	default:
		return strings.TrimSpace(freq)
	}
}

// SummarizeMultiTimeframeRows produces a short Chinese bias line.
func SummarizeMultiTimeframeRows(rows []PriceAnalysisRow) string {
	if len(rows) == 0 {
		return ""
	}
	byTF := map[string]string{}
	for _, r := range rows {
		byTF[r.Timeframe] = r.Conclusion
	}
	shortBull := bullishCount(byTF["5m"], byTF["60m"], byTF["日线"])
	shortBear := bearishCount(byTF["5m"], byTF["60m"], byTF["日线"])
	wk := byTF["周线"]
	switch {
	case shortBull >= 2 && (wk == RegimeDown || wk == RegimeWeakDown):
		return "短中线偏多，周线仍空。"
	case shortBear >= 2 && (wk == RegimeUp || wk == RegimeWeakUp):
		return "短中线偏空，周线仍多。"
	case shortBull >= 2 && wk == RegimeUp:
		return "多周期共振偏多。"
	case shortBear >= 2 && wk == RegimeDown:
		return "多周期共振偏空。"
	case byTF["日线"] == RegimeRange || byTF["60m"] == RegimeRange:
		return "短中线震荡，等待结构突破。"
	case byTF["日线"] == RegimeUnclear:
		return "日线结构不明确，谨慎对待信号。"
	default:
		return "多周期结论分化，需结合 Episode 与规则诊断。"
	}
}

func bullishCount(regs ...string) int {
	n := 0
	for _, r := range regs {
		if r == RegimeUp || r == RegimeWeakUp {
			n++
		}
	}
	return n
}

func bearishCount(regs ...string) int {
	n := 0
	for _, r := range regs {
		if r == RegimeDown || r == RegimeWeakDown {
			n++
		}
	}
	return n
}

// AppendPriceAnalysisMarkdown renders the multi-TF table for diagnose facts.
func AppendPriceAnalysisMarkdown(b *strings.Builder, pa *PriceAnalysis) {
	if pa == nil || len(pa.Rows) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### 多周期价格分析\n\n")
	if pa.SpotLabel != "" && pa.Code != "" {
		fmt.Fprintf(b, "**%s %s**\n\n", pa.Code, pa.SpotLabel)
	} else if pa.SpotLabel != "" {
		fmt.Fprintf(b, "**%s**\n\n", pa.SpotLabel)
	}
	fmt.Fprintf(b, "| 周期 | 结论 | 结构 | BOS/CHOCH | 位置 | ER | RVOL | 量能 |\n")
	fmt.Fprintf(b, "| --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, r := range pa.Rows {
		er := "-"
		if r.ER > 0 {
			er = fmt.Sprintf("%.2f", r.ER)
		}
		rvol := "-"
		if r.RVOL > 0 {
			rvol = fmt.Sprintf("%.2f", r.RVOL)
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s |\n",
			r.Timeframe, r.Conclusion, r.Structure, r.BosChoch, r.Position, er, rvol, r.VolumeNote)
	}
	if sr := strings.TrimSpace(pa.SupportResistance); sr != "" {
		fmt.Fprintf(b, "\n**支撑阻力**：%s\n", sr)
	}
	if s := strings.TrimSpace(pa.Summary); s != "" {
		fmt.Fprintf(b, "\n**综合**：%s\n", s)
	}
	if pa.PriceScore != 0 || strings.TrimSpace(pa.ScoreReason) != "" {
		if r := strings.TrimSpace(pa.ScoreReason); r != "" {
			fmt.Fprintf(b, "\n**多周期打分**：%.2f — %s\n", NormalizePriceScore(pa.PriceScore), r)
		} else {
			fmt.Fprintf(b, "\n**多周期打分**：%.2f\n", NormalizePriceScore(pa.PriceScore))
		}
	}
	if note := strings.TrimSpace(pa.FilterTimeframeNote); note != "" {
		fmt.Fprintf(b, "\n_%s_\n", note)
	}
}
