package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

// PriceAnalysisJEVChoice is one directional option with model probability.
type PriceAnalysisJEVChoice struct {
	Label       string  `json:"label"`
	Probability float64 `json:"probability"`
}

// PriceAnalysisJEV is the JEV 决策 directional read for lab price analysis.
type PriceAnalysisJEV struct {
	Source            string                   `json:"source"`
	Model             string                   `json:"model"`
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	RecommendedReason string                   `json:"recommended_reason"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	OK                bool                     `json:"ok"`
	Error             string                   `json:"error"`
}

var priceAnalysisJEVLabels = []string{"看多", "看空", "不明确"}

var priceAnalysisJEVCriteria = map[string]string{
	"看多": "多周期价格结构整体偏多，或关键周期结论一致指向多头",
	"看空": "多周期价格结构整体偏空，或关键周期结论一致指向空头",
	"不明确": "结构信息不足，或经权衡仍无法给出可靠方向（非仅因周期冲突就默认此项）",
}

// LabJEVKlineTaggedHit is one timeframe's pattern hits at the aligned anchor bar for JEV state.
type LabJEVKlineTaggedHit struct {
	Timeframe string
	Anchor    bool
	Hits      []slots.KlinePatternHit
}

// LabJEVKlineInput is kline pattern context for the single lab JEV decision at one anchor bar.
type LabJEVKlineInput struct {
	SignalFreq string
	TaggedHits []LabJEVKlineTaggedHit
	// DisplaySummary / DisplayBias may reflect MTF-weighted card fields (not a substitute for TaggedHits).
	DisplaySummary      string
	DisplayBias         *slots.KlinePatternBias
	MTFWeightedDisplay  bool
	Summary             string
	Bias                *slots.KlinePatternBias
	Hits                []slots.KlinePatternHit
}

// BuildPriceAnalysisJEVState compresses multi-TF PA into state for JEV 决策 (no kline patterns).
func BuildPriceAnalysisJEVState(pa *PriceAnalysis, signalSide string) string {
	var b strings.Builder
	if pa != nil {
		if c := strings.TrimSpace(pa.Code); c != "" {
			fmt.Fprintf(&b, "标的 %s\n", c)
		}
		if t := strings.TrimSpace(pa.AnchorTime); t != "" {
			fmt.Fprintf(&b, "锚点 %s 收盘 %.4g\n", t, pa.AnchorClose)
		} else if pa.SpotPrice != 0 {
			fmt.Fprintf(&b, "现价 %.4g\n", pa.SpotPrice)
		}
		if s := strings.TrimSpace(pa.Summary); s != "" {
			fmt.Fprintf(&b, "摘要：%s\n", s)
		}
		if sr := strings.TrimSpace(pa.SupportResistance); sr != "" {
			fmt.Fprintf(&b, "支撑阻力：%s\n", sr)
		}
		for _, row := range pa.Rows {
			tf := strings.TrimSpace(row.Timeframe)
			if tf == "" {
				continue
			}
			fmt.Fprintf(&b, "[%s] 结论=%s 结构=%s BOS/CHOCH=%s 位置=%s",
				tf, strings.TrimSpace(row.Conclusion), strings.TrimSpace(row.Structure),
				strings.TrimSpace(row.BosChoch), strings.TrimSpace(row.Position))
			if row.ER != 0 {
				fmt.Fprintf(&b, " ER=%.3f", row.ER)
			}
			if row.RVOL != 0 {
				fmt.Fprintf(&b, " RVOL=%.3f", row.RVOL)
			}
			if vn := strings.TrimSpace(row.VolumeNote); vn != "" {
				fmt.Fprintf(&b, " 量=%s", vn)
			}
			b.WriteByte('\n')
		}
	}
	side := strings.TrimSpace(signalSide)
	if side == "buy" {
		b.WriteString("信号侧：买入触发\n")
	} else if side == "sell" {
		b.WriteString("信号侧：卖出触发\n")
	}
	return strings.TrimSpace(b.String())
}

func appendKlinePatternJEVState(b *strings.Builder, kline *LabJEVKlineInput) {
	if kline == nil {
		return
	}
	summary := strings.TrimSpace(kline.DisplaySummary)
	if summary == "" {
		summary = strings.TrimSpace(kline.Summary)
	}
	bias := kline.DisplayBias
	if bias == nil {
		bias = kline.Bias
	}
	tagged := kline.TaggedHits
	hasLegacyHits := len(kline.Hits) > 0
	if summary == "" && bias == nil && len(tagged) == 0 && !hasLegacyHits {
		return
	}
	b.WriteString("\n--- 锚点 K 线形态（按周期标注，供与结构一并判断）---\n")
	if summary != "" {
		if kline.MTFWeightedDisplay {
			fmt.Fprintf(b, "形态展示摘要（多周期加权，仅供参考）：%s\n", summary)
		} else {
			fmt.Fprintf(b, "形态摘要：%s\n", summary)
		}
	}
	if bias != nil {
		if kline.MTFWeightedDisplay {
			fmt.Fprintf(b, "形态展示倾向（多周期加权，仅供参考）：%s", strings.TrimSpace(bias.Bias))
		} else {
			fmt.Fprintf(b, "形态倾向：%s", strings.TrimSpace(bias.Bias))
		}
		if bias.Score != 0 {
			fmt.Fprintf(b, "（得分 %.2f）", bias.Score)
		}
		b.WriteByte('\n')
	}
	for _, block := range tagged {
		for _, h := range block.Hits {
			appendLabJEVKlineHitLine(b, block.Timeframe, block.Anchor, h)
		}
	}
	if len(tagged) == 0 {
		for _, h := range kline.Hits {
			appendLabJEVKlineHitLine(b, labJEVTimeframeTag("", kline.SignalFreq), true, h)
		}
	}
}

func appendLabJEVKlineHitLine(b *strings.Builder, tfTag string, anchor bool, h slots.KlinePatternHit) {
	label := strings.TrimSpace(h.LabelCN)
	if label == "" {
		label = strings.TrimSpace(h.Name)
	}
	if label == "" {
		return
	}
	tfTag = strings.TrimSpace(tfTag)
	if tfTag == "" {
		tfTag = "信号"
	}
	prefix := tfTag + " 命中"
	if anchor {
		prefix = tfTag + " 锚点命中"
	}
	dir := klinePatternHitDirectionLabel(h.Direction)
	fmt.Fprintf(b, "%s：%s 方向=%s\n", prefix, label, dir)
}

func klinePatternHitDirectionLabel(direction int) string {
	switch direction {
	case 1:
		return "看多"
	case -1:
		return "看空"
	default:
		return "中性"
	}
}

func labJEVTimeframeTag(entryLabel, freq string) string {
	if l := strings.TrimSpace(entryLabel); l != "" {
		return l
	}
	switch NormalizeFrequency(freq) {
	case "5m":
		return "5m"
	case "60m":
		return "60m"
	case "daily":
		return "日线"
	case "weekly", "1wk":
		return "周线"
	default:
		if f := strings.TrimSpace(freq); f != "" {
			return f
		}
		return ""
	}
}

func labJEVHitsFromThreeBarAnchor(win *slots.KlinePatternThreeBarWindow) []slots.KlinePatternHit {
	if win == nil {
		return nil
	}
	for _, bar := range win.Bars {
		if bar.Role == "anchor" && !bar.Omitted && len(bar.KlinePatterns) > 0 {
			return bar.KlinePatterns
		}
	}
	return nil
}

func labJEVKlineTaggedHitsFromDetail(out *LabEpisodeDetail, signalFreq string) []LabJEVKlineTaggedHit {
	if out == nil {
		return nil
	}
	sigN := NormalizeFrequency(signalFreq)
	if mtf := out.KlinePatternMTF; mtf != nil {
		if sigN == "" {
			sigN = NormalizeFrequency(mtf.SignalFrequency)
		}
		var tagged []LabJEVKlineTaggedHit
		for _, tf := range mtf.Timeframes {
			hits := labJEVHitsFromThreeBarAnchor(tf.ThreeBar)
			if len(hits) == 0 {
				continue
			}
			isAnchor := NormalizeFrequency(tf.Frequency) == sigN
			tag := labJEVTimeframeTag(tf.Label, tf.Frequency)
			tagged = append(tagged, LabJEVKlineTaggedHit{
				Timeframe: tag,
				Anchor:    isAnchor,
				Hits:      hits,
			})
		}
		return tagged
	}
	if win := out.KlinePatternThreeBar; win != nil {
		if hits := labJEVHitsFromThreeBarAnchor(win); len(hits) > 0 {
			tag := labJEVTimeframeTag("", sigN)
			if tag == "" {
				tag = "信号"
			}
			return []LabJEVKlineTaggedHit{{Timeframe: tag, Anchor: true, Hits: hits}}
		}
	}
	if len(out.KlinePatterns) > 0 {
		tag := labJEVTimeframeTag("", sigN)
		if tag == "" {
			tag = "信号"
		}
		return []LabJEVKlineTaggedHit{{Timeframe: tag, Anchor: true, Hits: out.KlinePatterns}}
	}
	return nil
}

func labJEVKlineFromDetail(out *LabEpisodeDetail) *LabJEVKlineInput {
	if out == nil {
		return nil
	}
	sigFreq := ""
	if out.KlinePatternMTF != nil {
		sigFreq = out.KlinePatternMTF.SignalFrequency
	}
	in := &LabJEVKlineInput{
		SignalFreq:         sigFreq,
		DisplaySummary:     patternSummaryText(out),
		DisplayBias:        patternBiasForDetail(out),
		MTFWeightedDisplay: out.KlinePatternMTF != nil && strings.TrimSpace(out.KlinePatternMTF.WeightSchema) != "",
		TaggedHits:         labJEVKlineTaggedHitsFromDetail(out, sigFreq),
	}
	if in.DisplaySummary == "" && in.DisplayBias == nil && len(in.TaggedHits) == 0 {
		return nil
	}
	return in
}

func labJEVKlineInputEmpty(kline *LabJEVKlineInput) bool {
	if kline == nil {
		return true
	}
	if len(kline.TaggedHits) > 0 || len(kline.Hits) > 0 {
		return false
	}
	if strings.TrimSpace(kline.Summary) != "" || strings.TrimSpace(kline.DisplaySummary) != "" {
		return false
	}
	if kline.Bias != nil || kline.DisplayBias != nil {
		return false
	}
	return true
}

func labPAInformative(pa *PriceAnalysis) bool {
	if pa == nil {
		return false
	}
	return strings.TrimSpace(pa.Summary) != "" || len(pa.Rows) > 0
}

// ResolvePriceAnalysisJEV calls JEV Choice once; never falls back to local PA rules.
func ResolvePriceAnalysisJEV(ctx context.Context, jev *JevClient, pa *PriceAnalysis, signalSide string) PriceAnalysisJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyPriceAnalysisJEV(model)
	if !labPAInformative(pa) {
		base.Error = "lab decision state empty"
		return base
	}
	state := BuildPriceAnalysisJEVState(pa, signalSide)
	if state == "" {
		base.Error = "lab decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中的多周期价格结构（结论、摆动结构、BOS/CHOCH、区间位置、ER、RVOL），" +
		"对该信号上下文应如何判断价格方向？只能从给定选项中选一项。" +
		"规则：①多周期结构同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅部分周期有信息则跟随有信息的一方；④信息均不足或确实无法判断时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, priceAnalysisJEVCriteria)
	if err != nil {
		base.Error = err.Error()
		return base
	}
	if respModel != "" {
		base.Model = respModel
	}
	idx, ok := indexPriceAnalysisJEVLabel(choice)
	if !ok {
		base.Error = fmt.Sprintf("unexpected jev choice: %q", choice)
		return base
	}
	base.OK = true
	base.Error = ""
	base.RecommendedChoice = choice
	base.RecommendedIndex = idx
	base.Choices = priceAnalysisJEVChoicesFromProbs(probs)
	base.RecommendedReason = labJEVReasonFromSummary(ctx, jev, choice, idx, base.Choices, confidence, state)
	return base
}

func emptyPriceAnalysisJEV(model string) PriceAnalysisJEV {
	return PriceAnalysisJEV{
		Source:  "jev",
		Model:   model,
		Choices: priceAnalysisJEVChoicesFromProbs(nil),
		OK:      false,
	}
}

func priceAnalysisJEVChoicesFromProbs(probs map[string]float64) []PriceAnalysisJEVChoice {
	out := make([]PriceAnalysisJEVChoice, 0, len(priceAnalysisJEVLabels))
	for _, label := range priceAnalysisJEVLabels {
		p := 0.0
		if probs != nil {
			p = probs[label]
		}
		out = append(out, PriceAnalysisJEVChoice{Label: label, Probability: p})
	}
	return out
}

func indexPriceAnalysisJEVLabel(label string) (int, bool) {
	label = strings.TrimSpace(label)
	for i, l := range priceAnalysisJEVLabels {
		if l == label {
			return i, true
		}
	}
	return 0, false
}

func applyPriceAnalysisJEV(ctx context.Context, out *LabEpisodeDetail, side string, jev *JevClient) {
	if out == nil || out.PriceAnalysis == nil || !labPAInformative(out.PriceAnalysis) {
		return
	}
	rec := ResolvePriceAnalysisJEV(ctx, jev, out.PriceAnalysis, side)
	out.PriceAnalysisJEV = &rec
	if out.PriceAnalysis != nil {
		out.PriceAnalysis.PriceAnalysisJEV = &rec
	}
}
