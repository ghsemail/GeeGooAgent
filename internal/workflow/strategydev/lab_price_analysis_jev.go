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

// PriceAnalysisJEV is the System One (JEV) directional read for lab price analysis.
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
	"看多": "多周期价格结构与锚点 K 线形态整体偏多，或仅有一方明确偏多且另一方中性/缺失",
	"看空": "多周期价格结构与锚点 K 线形态整体偏空，或仅有一方明确偏空且另一方中性/缺失",
	"不明确": "结构与形态均信息不足，或经权衡仍无法给出可靠方向（非仅因两方冲突就默认此项）",
}

// LabJEVKlineInput is kline pattern context for the single lab JEV decision at one anchor bar.
type LabJEVKlineInput struct {
	Summary string
	Bias    *slots.KlinePatternBias
	Hits    []slots.KlinePatternHit
}

// BuildPriceAnalysisJEVState compresses multi-TF PA and anchor-bar kline patterns into state for System One.
func BuildPriceAnalysisJEVState(pa *PriceAnalysis, signalSide string, kline *LabJEVKlineInput) string {
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
		if pa.PriceScore != 0 || strings.TrimSpace(pa.ScoreReason) != "" {
			fmt.Fprintf(&b, "综合分 %.2f", pa.PriceScore)
			if sr := strings.TrimSpace(pa.ScoreReason); sr != "" {
				fmt.Fprintf(&b, " · %s", sr)
			}
			b.WriteByte('\n')
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
		for label, br := range pa.TFScores {
			if br.Missing {
				continue
			}
			fmt.Fprintf(&b, "分项[%s] 子分=%.3f 结论=%.3f 位置=%.3f BOS=%.3f\n",
				label, br.Subscore, br.Conclusion, br.Position, br.BosChoch)
		}
	}
	side := strings.TrimSpace(signalSide)
	if side == "buy" {
		b.WriteString("信号侧：买入触发\n")
	} else if side == "sell" {
		b.WriteString("信号侧：卖出触发\n")
	}
	appendKlinePatternJEVState(&b, kline)
	return strings.TrimSpace(b.String())
}

func appendKlinePatternJEVState(b *strings.Builder, kline *LabJEVKlineInput) {
	if kline == nil {
		return
	}
	summary := strings.TrimSpace(kline.Summary)
	hasHits := len(kline.Hits) > 0
	if summary == "" && !hasHits && kline.Bias == nil {
		return
	}
	b.WriteString("\n--- 锚点 K 线形态（本地规则命中，供与结构一并判断）---\n")
	if summary != "" {
		fmt.Fprintf(b, "形态摘要：%s\n", summary)
	}
	if kline.Bias != nil {
		fmt.Fprintf(b, "形态倾向：%s", strings.TrimSpace(kline.Bias.Bias))
		if kline.Bias.Score != 0 {
			fmt.Fprintf(b, "（得分 %.2f）", kline.Bias.Score)
		}
		b.WriteByte('\n')
	}
	for _, h := range kline.Hits {
		dir := "中性"
		switch h.Direction {
		case 1:
			dir = "看多"
		case -1:
			dir = "看空"
		}
		label := strings.TrimSpace(h.LabelCN)
		if label == "" {
			label = strings.TrimSpace(h.Name)
		}
		if label == "" {
			continue
		}
		fmt.Fprintf(b, "命中：%s 方向=%s\n", label, dir)
	}
}

func labJEVKlineFromDetail(out *LabEpisodeDetail) *LabJEVKlineInput {
	if out == nil {
		return nil
	}
	in := &LabJEVKlineInput{
		Summary: patternSummaryText(out),
		Bias:    patternBiasForDetail(out),
		Hits:    out.KlinePatterns,
	}
	if in.Summary == "" && in.Bias == nil && len(in.Hits) == 0 {
		return nil
	}
	return in
}

func labJEVKlineInputEmpty(kline *LabJEVKlineInput) bool {
	return kline == nil
}

func labPAInformative(pa *PriceAnalysis) bool {
	if pa == nil {
		return false
	}
	if strings.TrimSpace(pa.Summary) != "" || len(pa.Rows) > 0 {
		return true
	}
	if pa.PriceScore != 0 || strings.TrimSpace(pa.ScoreReason) != "" {
		return true
	}
	return false
}

// ResolvePriceAnalysisJEV calls System One choice primitive; never falls back to local PA/pattern rules.
func ResolvePriceAnalysisJEV(ctx context.Context, jev *JevClient, pa *PriceAnalysis, signalSide string, kline *LabJEVKlineInput) PriceAnalysisJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyPriceAnalysisJEV(model)
	if !labPAInformative(pa) && labJEVKlineInputEmpty(kline) {
		base.Error = "lab decision state empty"
		return base
	}
	state := BuildPriceAnalysisJEVState(pa, signalSide, kline)
	if state == "" {
		base.Error = "lab decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中的多周期价格结构（结论、摆动结构、BOS/CHOCH、区间位置、ER、RVOL、综合分）" +
		"与锚点 K 线形态命中/倾向，对该信号上下文应如何判断价格方向？只能从给定选项中选一项。" +
		"规则：①结构与形态同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅结构或仅形态一方有信息则跟随有信息的一方；④两方均不足或确实无法判断时才选不明确。"
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
	if confidence > 0 {
		base.RecommendedReason = fmt.Sprintf("结构+形态 JEV 倾向%s（置信 %.0f%%）", choice, confidence*100)
	} else if p := base.Choices[idx].Probability; p > 0 {
		base.RecommendedReason = fmt.Sprintf("结构+形态 JEV 倾向%s（%.0f%%）", choice, p*100)
	} else {
		base.RecommendedReason = fmt.Sprintf("结构+形态 JEV 倾向%s", choice)
	}
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
	if out == nil {
		return
	}
	kline := labJEVKlineFromDetail(out)
	if out.PriceAnalysis == nil && labJEVKlineInputEmpty(kline) {
		return
	}
	rec := ResolvePriceAnalysisJEV(ctx, jev, out.PriceAnalysis, side, kline)
	out.PriceAnalysisJEV = &rec
	if out.PriceAnalysis != nil {
		out.PriceAnalysis.PriceAnalysisJEV = &rec
	}
}
