package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// KlineJEV is the JEV 决策 directional read for lab K-line pattern hits at the anchor bar.
type KlineJEV struct {
	Source            string                   `json:"source"`
	Model             string                   `json:"model"`
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	RecommendedReason string                   `json:"recommended_reason"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	OK                bool                     `json:"ok"`
	Error             string                   `json:"error"`
}

var klineJEVCriteria = map[string]string{
	"看多": "锚点及多周期 K 线形态命中/倾向整体偏多，或仅有一方明确偏多且另一方中性/缺失",
	"看空": "锚点及多周期 K 线形态命中/倾向整体偏空，或仅有一方明确偏空且另一方中性/缺失",
	"不明确": "形态信息不足，或经权衡仍无法给出可靠方向（非仅因多方冲突就默认此项）",
}

// BuildKlineJEVState compresses anchor-bar kline pattern hits into state for JEV 决策.
func BuildKlineJEVState(kline *LabJEVKlineInput, signalSide string, longOnly bool) string {
	if labJEVKlineInputEmpty(kline) {
		return ""
	}
	var b strings.Builder
	if heading := LabSignalSideTriggerHeading(signalSide, longOnly); heading != "" {
		b.WriteString(heading)
		b.WriteByte('\n')
	}
	appendKlinePatternJEVState(&b, kline)
	return strings.TrimSpace(b.String())
}

// ResolveKlineJEV calls JEV Choice once; never falls back to local pattern rules.
func ResolveKlineJEV(ctx context.Context, jev *JevClient, kline *LabJEVKlineInput, signalSide string, longOnly bool) KlineJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyKlineJEV(model)
	if labJEVKlineInputEmpty(kline) {
		base.Error = "lab kline decision state empty"
		return base
	}
	state := BuildKlineJEVState(kline, signalSide, longOnly)
	if state == "" {
		base.Error = "lab kline decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中的锚点 K 线形态命中、摘要与倾向，对该信号上下文应如何判断价格方向？只能从给定选项中选一项。" +
		"规则：①多周期形态同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅摘要或仅命中一方有信息则跟随有信息的一方；④信息均不足或确实无法判断时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, klineJEVCriteria)
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
	base.RecommendedReason = labKlineJEVReasonFromSummary(ctx, jev, choice, idx, base.Choices, confidence, state)
	return base
}

func emptyKlineJEV(model string) KlineJEV {
	return KlineJEV{
		Source:  "jev",
		Model:   model,
		Choices: priceAnalysisJEVChoicesFromProbs(nil),
		OK:      false,
	}
}

func applyKlineJEV(ctx context.Context, out *LabEpisodeDetail, side string, jev *JevClient, longOnly bool) {
	if out == nil {
		return
	}
	kline := labJEVKlineFromDetail(out)
	if labJEVKlineInputEmpty(kline) {
		return
	}
	rec := ResolveKlineJEV(ctx, jev, kline, side, longOnly)
	out.KlineJEV = &rec
}

func labKlineJEVReasonFromSummary(ctx context.Context, jev *JevClient, choice string, idx int, choices []PriceAnalysisJEVChoice, confidence float64, facts string) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizeKlineJEVReason(ctx, choice, idx, choices, confidence, facts)
	if err != nil {
		return labJEVReasonSummaryFailed(err.Error())
	}
	return reason
}
