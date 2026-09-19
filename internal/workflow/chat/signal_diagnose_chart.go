package chat

import (
	"fmt"
	"math"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// chartProbePayload is the probe snapshot for StrategyKlineChart mirroring.
func chartProbePayload(flow *Flow) map[string]any {
	if flow == nil || flow.ProbeRaw == nil {
		return nil
	}
	out := map[string]any{
		"code":        flow.StockCode,
		"frequency":   probeFrequency(flow),
		"months_back": flow.MonthsBack,
		"bars":        flow.ProbeRaw["bars"],
		"buy_merged":  flow.ProbeRaw["buy_merged"],
		"sell_merged": flow.ProbeRaw["sell_merged"],
	}
	if len(flow.SignalEval.BuyDetails) > 0 || len(flow.SignalEval.SellDetails) > 0 {
		out["episodes"] = episodeOverlayPayload(flow.SignalEval)
	}
	return out
}

func probeFrequency(flow *Flow) string {
	if len(flow.Strategies) > 0 && strings.TrimSpace(flow.Strategies[0].Frequency) != "" {
		return flow.Strategies[0].Frequency
	}
	return "60m"
}

func episodeOverlayPayload(eval slots.SignalEpisodeEval) []map[string]any {
	out := make([]map[string]any, 0, len(eval.BuyDetails)+len(eval.SellDetails))
	for i, ep := range eval.BuyDetails {
		out = append(out, episodeOverlayRow("buy", i+1, ep))
	}
	for i, ep := range eval.SellDetails {
		out = append(out, episodeOverlayRow("sell", i+1, ep))
	}
	return out
}

func episodeOverlayRow(side string, idx int, ep slots.SignalEpisodeDetail) map[string]any {
	endIdx, endTime, endClose, hit, complete := ep.EndIdx, ep.EndTime, ep.EndClose, ep.Hit, ep.Complete
	if !complete && ep.PathEvaluable {
		endIdx = ep.PathEndIdx
		endTime = ep.PathEndTime
		endClose = ep.PathEndClose
		hit = ep.PathHit
	}
	row := map[string]any{
		"side":             side,
		"index":            idx,
		"start_idx":        ep.StartIdx,
		"end_idx":          endIdx,
		"start_time":       ep.StartTime,
		"end_time":         endTime,
		"start_close":      ep.StartClose,
		"end_close":        endClose,
		"hit":              hit,
		"complete":         complete || ep.PathEvaluable,
		"holding_bars":     ep.PathHoldingBars,
		"direction_return": ep.PathDirectionReturn,
		"path_end_reason":  ep.PathEndReason,
		"peak_return":      ep.PeakReturn,
		"max_drawdown_from_peak": ep.MaxDrawdownFromPeak,
		"path_evaluable":   ep.PathEvaluable,
	}
	if ep.Complete {
		row["strict_complete"] = true
		row["direction_return"] = ep.DirectionReturn
		row["holding_bars"] = ep.HoldingBars
	}
	if ep.OppositeTime != "" {
		row["opposite_time"] = ep.OppositeTime
	}
	return row
}

func appendKlineOverviewSection(b *strings.Builder, flow *Flow) {
	if flow == nil || flow.ProbeRaw == nil {
		return
	}
	bars, _ := flow.ProbeRaw["bars"].([]any)
	if len(bars) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### K 线概览（收盘价走势 + 信号/ Episode）\n\n")
	fmt.Fprintf(b, "_Chat 内为缩略走势；策略开发页可渲染完整交互 K 线（见下方对接设计）。_\n\n")
	if line := closingPriceSparkline(bars, 72); line != "" {
		fmt.Fprintf(b, "```\n%s\n```\n\n", line)
	}
	if legend := sparklineLegend(bars); legend != "" {
		fmt.Fprintf(b, "%s\n\n", legend)
	}
	if timeline := episodeMermaidTimeline(flow.SignalEval); timeline != "" {
		fmt.Fprintf(b, "**Episode 持有区间（时间轴）**\n\n```mermaid\n%s\n```\n\n", timeline)
	}
	fmt.Fprintf(b, "> 图例：▲ 买信号  ▼ 卖信号  │ 闭合买 Episode（绿=命中/红=未中）  ┆ 闭合卖 Episode\n\n")
	b.WriteString(signalMarkerStrip(bars, flow.ProbeRaw, flow.SignalEval))
}

func closingPriceSparkline(bars []any, width int) string {
	closes := make([]float64, 0, len(bars))
	for _, raw := range bars {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c := slotsFloat(row["close"])
		if c > 0 {
			closes = append(closes, c)
		}
	}
	if len(closes) < 2 || width < 8 {
		return ""
	}
	sampled := downsampleFloats(closes, width)
	minV, maxV := sampled[0], sampled[0]
	for _, v := range sampled[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if math.Abs(maxV-minV) < 1e-9 {
		maxV = minV + 1
	}
	ramp := "▁▂▃▄▅▆▇█"
	var b strings.Builder
	for _, v := range sampled {
		idx := int(math.Round((v - minV) / (maxV - minV) * float64(len(ramp)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(ramp) {
			idx = len(ramp) - 1
		}
		b.WriteByte(ramp[idx])
	}
	return fmt.Sprintf("%s  (%.1f → %.1f)", b.String(), minV, maxV)
}

func sparklineLegend(bars []any) string {
	if len(bars) == 0 {
		return ""
	}
	first, last := barTimeAt(bars, 0), barTimeAt(bars, len(bars)-1)
	return fmt.Sprintf("时间范围：%s → %s（共 %d 根 K 线）", dashTime(first), dashTime(last), len(bars))
}

func signalMarkerStrip(bars []any, probeRaw map[string]any, eval slots.SignalEpisodeEval) string {
	if len(bars) == 0 {
		return ""
	}
	width := 72
	buyMerged, _ := probeRaw["buy_merged"].([]any)
	sellMerged, _ := probeRaw["sell_merged"].([]any)
	closes := make([]float64, len(bars))
	for i, raw := range bars {
		row, _ := raw.(map[string]any)
		closes[i] = slotsFloat(row["close"])
	}
	sigLine := make([]rune, width)
	epLine := make([]rune, width)
	for j := 0; j < width; j++ {
		sigLine[j] = '·'
		epLine[j] = ' '
	}
	for i := 0; i < len(bars); i++ {
		col := idxToCol(i, len(bars), width)
		if i < len(buyMerged) && mergedVal(buyMerged[i]) == 1 {
			sigLine[col] = '▲'
		}
		if i < len(sellMerged) && mergedVal(sellMerged[i]) == -1 {
			if sigLine[col] == '·' {
				sigLine[col] = '▼'
			}
		}
	}
	paintEpisodes(epLine, eval.BuyDetails, len(bars), width, '│', true)
	paintEpisodes(epLine, eval.SellDetails, len(bars), width, '┆', false)
	return fmt.Sprintf("```\n信号  %s\nEpisode %s\n```\n", string(sigLine), string(epLine))
}

func paintEpisodes(line []rune, details []slots.SignalEpisodeDetail, barCount, width int, fill rune, _ bool) {
	for _, ep := range details {
		end, ok := episodeSpanEnd(ep)
		if !ok || end < ep.StartIdx {
			continue
		}
		c0 := idxToCol(ep.StartIdx, barCount, width)
		c1 := idxToCol(end, barCount, width)
		for c := c0; c <= c1 && c < len(line); c++ {
			if line[c] == ' ' {
				line[c] = fill
			}
		}
	}
}

func idxToCol(idx, barCount, width int) int {
	if barCount <= 1 {
		return 0
	}
	col := int(math.Round(float64(idx) / float64(barCount-1) * float64(width-1)))
	if col < 0 {
		return 0
	}
	if col >= width {
		return width - 1
	}
	return col
}

func downsampleFloats(values []float64, width int) []float64 {
	if len(values) <= width {
		return values
	}
	out := make([]float64, width)
	for i := 0; i < width; i++ {
		idx := int(math.Round(float64(i) / float64(width-1) * float64(len(values)-1)))
		out[i] = values[idx]
	}
	return out
}

func barTimeAt(bars []any, i int) string {
	if i < 0 || i >= len(bars) {
		return ""
	}
	row, _ := bars[i].(map[string]any)
	return strings.TrimSpace(fmt.Sprint(row["time"]))
}

func mergedVal(raw any) int {
	switch v := raw.(type) {
	case int:
	 return v
	case float64:
		return int(v)
	default:
		return 0
	}
}

func slotsFloat(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}

func episodeMermaidTimeline(eval slots.SignalEpisodeEval) string {
	if len(eval.BuyDetails) == 0 && len(eval.SellDetails) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("gantt\n    dateFormat YYYY-MM-DD HH:mm\n    axisFormat %m-%d\n")
	b.WriteString("    title Episode 持有区间（买/卖交替）\n")
	appendMermaidEpisodes(&b, "买入", eval.BuyDetails)
	appendMermaidEpisodes(&b, "卖出", eval.SellDetails)
	return strings.TrimSpace(b.String())
}

func episodeSpanEnd(ep slots.SignalEpisodeDetail) (int, bool) {
	if ep.Complete && ep.EndIdx >= ep.StartIdx {
		return ep.EndIdx, true
	}
	if ep.PathEvaluable && ep.PathEndIdx >= ep.StartIdx {
		return ep.PathEndIdx, true
	}
	return -1, false
}

func appendMermaidEpisodes(b *strings.Builder, section string, details []slots.SignalEpisodeDetail) {
	if len(details) == 0 {
		return
	}
	fmt.Fprintf(b, "    section %s\n", section)
	for i, ep := range details {
		_, ok := episodeSpanEnd(ep)
		if !ok {
			fmt.Fprintf(b, "    E%d 未闭合 :crit, %s, 1h\n", i+1, mermaidTime(ep.StartTime))
			continue
		}
		endTime := ep.EndTime
		hit := ep.Hit
		if !ep.Complete && ep.PathEvaluable {
			endTime = ep.PathEndTime
			hit = ep.PathHit
		}
		status := "done"
		if !hit {
			status = "crit"
		}
		fmt.Fprintf(b, "    E%d %s :%s, %s, %s\n", i+1, mermaidEpLabel(ep), status, mermaidTime(ep.StartTime), mermaidTime(endTime))
	}
}

func mermaidTime(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.ReplaceAll(raw, " ", " ")
	if len(raw) >= 16 {
		return raw[:16]
	}
	return raw
}

func mermaidEpLabel(ep slots.SignalEpisodeDetail) string {
	return fmt.Sprintf("%+.1f%% %dk", ep.DirectionReturn*100, ep.HoldingBars)
}
