package diagnoseskill

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func populateDetailFromLanes(bundle LaneBundle, macro bool) (*strategydev.LabEpisodeDetail, string, float64, float64, error) {
	out := &strategydev.LabEpisodeDetail{}
	if macro {
		out.KlineJEVProfile = strategydev.LabKlineJEVProfileDailyMacro
	}
	signalFreq := strings.TrimSpace(bundle.SignalFrequency)
	paRows, err := laneArray(bundle.PriceAnalysis, "price_analysis")
	if err != nil {
		return nil, "", 0, 0, err
	}
	klineRows, _ := laneArray(bundle.KlinePattern, "kline_pattern")
	taRows, _ := laneArray(bundle.TechnicalAnalysis, "technical_analysis")
	itRows, _ := laneArray(bundle.IndexTrend, "index_trend")

	pa := convertPriceAnalysis(bundle.Code, paRows, bundle.SpotPrice)
	out.PriceAnalysis = pa

	if len(klineRows) > 0 {
		populateKlineDetail(out, klineRows, signalFreq, macro)
	}

	ta := convertTechnicalAnalysis(bundle.Code, taRows)
	if ta != nil {
		out.TechnicalAnalysis = ta
	}
	it := convertIndexTrend(bundle.Code, itRows)
	if it != nil {
		out.IndexTrend = it
	}

	anchorFreq := signalFreq
	if macro {
		anchorFreq = "daily"
	}
	timeS, closeP := anchorFromRows(klineRows, paRows, anchorFreq)
	return out, timeS, closeP, bundle.SpotPrice, nil
}

func convertPriceAnalysis(code string, rows []map[string]any, spot float64) *strategydev.PriceAnalysis {
	if len(rows) == 0 {
		return nil
	}
	pa := &strategydev.PriceAnalysis{Code: code, SpotPrice: spot}
	for _, row := range rows {
		tf, _ := row["frequency"].(string)
		structBits, _ := row["structure"].([]any)
		var parts []string
		for _, b := range structBits {
			if s, ok := b.(string); ok {
				parts = append(parts, s)
			}
		}
		structure := strings.Join(parts, "+")
		if structure == "" && len(parts) > 0 {
			structure = strings.Join(parts, "")
		}
		bos := structureHits(row, "BOS", "CHOCH")
		pos := ""
		if rp, ok := row["range_position"].(float64); ok {
			pos = fmt.Sprintf("%.2f", rp)
		}
		conclusion := structure
		r := strategydev.PriceAnalysisRow{
			Timeframe:  tf,
			Conclusion: conclusion,
			Structure:  structure,
			BosChoch:   bos,
			Position:   pos,
		}
		if er, ok := row["er"].(float64); ok {
			r.ER = er
		}
		if rv, ok := row["rvol"].(float64); ok {
			r.RVOL = rv
		}
		if vr, ok := row["volume_ratio"].(float64); ok && vr > 0 {
			r.VolumeNote = fmt.Sprintf("volume_ratio=%.3f", vr)
		}
		pa.Rows = append(pa.Rows, r)
	}
	if len(pa.Rows) > 0 {
		pa.Summary = pa.Rows[0].Conclusion
	}
	return pa
}

func structureHits(row map[string]any, names ...string) string {
	hits, _ := row["hits"].([]any)
	var out []string
	for _, h := range hits {
		m, _ := h.(map[string]any)
		name, _ := m["name"].(string)
		for _, want := range names {
			if strings.EqualFold(name, want) {
				out = append(out, name)
			}
		}
	}
	return strings.Join(out, "/")
}

func convertTechnicalAnalysis(code string, rows []map[string]any) *strategydev.LabTechnicalAnalysis {
	if len(rows) == 0 {
		return nil
	}
	ta := &strategydev.LabTechnicalAnalysis{
		Code: code, StrategyVersion: "alpha", OK: true,
		Timeframes: map[string]strategydev.LabTechnicalAnalysisTF{},
	}
	for _, row := range rows {
		freq, _ := row["frequency"].(string)
		key := strategydev.NormalizeFrequency(freq)
		tf := strategydev.LabTechnicalAnalysisTF{Frequency: freq, OK: true}
		tf.Indicators = indicatorsFromDiagnoseRow(row)
		if t, ok := row["time"].(string); ok {
			tf.LastBarTime = t
			if ta.AnchorTime == "" {
				ta.AnchorTime = t
			}
		}
		ta.Timeframes[key] = tf
	}
	return ta
}

func indicatorsFromDiagnoseRow(row map[string]any) signalclient.TechnicalAnalysisIndicators {
	var ind signalclient.TechnicalAnalysisIndicators
	if ema, ok := row["ema"].(map[string]any); ok {
		ind.EMA.EMA13, _ = ema["ema_13"].(float64)
		ind.EMA.EMA26, _ = ema["ema_26"].(float64)
		ind.EMA.EMA13Slope, _ = ema["ema_13_slope"].(float64)
		ind.EMA.EMA26Slope, _ = ema["ema_26_slope"].(float64)
	}
	if env, ok := row["envelope"].(map[string]any); ok {
		ind.Envelope.MidEMA22, _ = env["mid_ema_22"].(float64)
		ind.Envelope.Upper, _ = env["upper"].(float64)
		ind.Envelope.Lower, _ = env["lower"].(float64)
		ind.Envelope.HalfWidth, _ = env["half_width"].(float64)
	}
	if macd, ok := row["macd"].(map[string]any); ok {
		ind.MACD.Line, _ = macd["line"].(float64)
		ind.MACD.Signal, _ = macd["signal"].(float64)
		ind.MACD.Histogram, _ = macd["histogram"].(float64)
		ind.MACD.HistogramSlope, _ = macd["histogram_slope"].(float64)
	}
	if st, ok := row["stochastic"].(map[string]any); ok {
		ind.Stochastic.SlowK, _ = st["slow_k"].(float64)
		ind.Stochastic.SlowD, _ = st["slow_d"].(float64)
	}
	if rsi, ok := row["rsi"].(map[string]any); ok {
		ind.RSI.Value, _ = rsi["value"].(float64)
	}
	if fi, ok := row["force_index"].(map[string]any); ok {
		ind.ForceIndex.EMA2, _ = fi["ema_2"].(float64)
		ind.ForceIndex.EMA13, _ = fi["ema_13"].(float64)
	}
	return ind
}

func convertIndexTrend(code string, rows []map[string]any) *strategydev.LabIndexTrend {
	if len(rows) == 0 {
		return nil
	}
	it := &strategydev.LabIndexTrend{
		Code: code, StrategyVersion: "alpha", OK: true,
		Timeframes: map[string]strategydev.LabIndexTrendTF{},
	}
	for _, row := range rows {
		freq, _ := row["frequency"].(string)
		key := strategydev.NormalizeFrequency(freq)
		tf := strategydev.LabIndexTrendTF{Frequency: freq, OK: true}
		if t, ok := row["time"].(string); ok {
			tf.LastBarTime = t
			if it.AnchorTime == "" {
				it.AnchorTime = t
			}
		}
		hits, _ := row["hits"].([]any)
		var longN, shortN, flatN int
		for _, h := range hits {
			m, _ := h.(map[string]any)
			name, _ := m["name"].(string)
			sig := intFromAny(m["signal"])
			stance := indexStance(sig)
			if stance == "long" {
				longN++
			} else if stance == "short" {
				shortN++
			} else {
				flatN++
			}
			tf.Trends = append(tf.Trends, signalclient.IndexTrendTrendRow{
				Name: name, Index: name, Stance: stance, Value: sig,
			})
		}
		tf.Summary = signalclient.IndexTrendSummary{
			Long: longN, Short: shortN, Flat: flatN, Net: longN - shortN,
		}
		it.Timeframes[key] = tf
	}
	return it
}

func indexStance(signal int) string {
	switch signal {
	case 1:
		return "long"
	case -1:
		return "short"
	default:
		return "flat"
	}
}

func populateKlineDetail(out *strategydev.LabEpisodeDetail, rows []map[string]any, signalFreq string, macro bool) {
	var tagged []strategydev.LabJEVKlineTaggedHit
	var anchorHits []slots.KlinePatternHit
	for _, row := range rows {
		freq, _ := row["frequency"].(string)
		hits := patternHitsFromRow(row)
		anchor := strategydev.NormalizeFrequency(freq) == strategydev.NormalizeFrequency(signalFreq)
		if macro {
			anchor = strategydev.NormalizeFrequency(freq) == "daily"
		}
		if len(hits) > 0 {
			tagged = append(tagged, strategydev.LabJEVKlineTaggedHit{
				Timeframe: freq, Anchor: anchor, Hits: hits,
			})
		}
		if anchor {
			anchorHits = hits
		}
	}
	summary := slots.KlinePatternSummary(anchorHits)
	out.KlinePatterns = anchorHits
	out.KlinePatternSummary = summary
	if macro && len(rows) > 0 {
		row := rows[0]
		if len(rows) > 1 {
			for _, r := range rows {
				if f, _ := r["frequency"].(string); strategydev.NormalizeFrequency(f) == "daily" {
					row = r
					break
				}
			}
		}
		out.KlinePatternThreeBar = threeBarFromRow(row, anchorHits)
		out.KlineJEVProfile = strategydev.LabKlineJEVProfileDailyMacro
	}
	if len(tagged) > 0 {
		out.KlinePattern = &strategydev.LabKlinePatternDetail{
			Summary: summary, Patterns: anchorHits,
			ThreeBar: out.KlinePatternThreeBar,
		}
	}
	_ = tagged // tagged hits wired via KlinePattern fields for labJEVKlineFromDetail
}

func patternHitsFromRow(row map[string]any) []slots.KlinePatternHit {
	hits, _ := row["hits"].([]any)
	var out []slots.KlinePatternHit
	for _, h := range hits {
		m, _ := h.(map[string]any)
		name, _ := m["name"].(string)
		dir := intFromAny(m["signal"])
		if dir == 0 {
			dir = intFromAny(m["direction"])
		}
		out = append(out, slots.KlinePatternHit{Name: name, Direction: dir})
	}
	return out
}

func threeBarFromRow(row map[string]any, hits []slots.KlinePatternHit) *slots.KlinePatternThreeBarWindow {
	o, _ := row["open"].(float64)
	h, _ := row["high"].(float64)
	l, _ := row["low"].(float64)
	c, _ := row["close"].(float64)
	t, _ := row["time"].(string)
	if c == 0 && o == 0 {
		return nil
	}
	return &slots.KlinePatternThreeBarWindow{
		Bars: []slots.KlinePatternBarWindowEntry{{
			Role: "anchor", Time: t, Open: o, High: h, Low: l, Close: c,
			KlinePatterns: hits,
		}},
	}
}

func anchorFromRows(klineRows, paRows []map[string]any, anchorFreq string) (time string, close float64) {
	anchorFreq = strategydev.NormalizeFrequency(anchorFreq)
	for _, rows := range [][]map[string]any{klineRows, paRows} {
		for _, row := range rows {
			f, _ := row["frequency"].(string)
			if strategydev.NormalizeFrequency(f) != anchorFreq {
				continue
			}
			if t, ok := row["time"].(string); ok {
				time = t
			}
			if c, ok := row["close"].(float64); ok {
				close = c
			}
			if time != "" || close != 0 {
				return time, close
			}
		}
	}
	return "", 0
}

func intFromAny(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func rawLaneArray(raw json.RawMessage, key string) json.RawMessage {
	var doc map[string]json.RawMessage
	if json.Unmarshal(raw, &doc) != nil {
		return nil
	}
	return doc[key]
}