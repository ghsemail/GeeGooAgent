package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

var indexTrendJEVCriteria = map[string]string{
	"看多": "ADX / EMA(中长期) / SAR 三旗在信号周期与更高周期上整体偏多，或冲突时经权衡仍偏多",
	"看空": "三旗整体偏空，或冲突时经权衡仍偏空",
	"不明确": "多周期三旗严重冲突、多数为平、或信息不足以可靠定向",
}

// BuildIndexTrendJEVState compresses multi-TF trend flags into state for JEV 决策.
func BuildIndexTrendJEVState(it *LabIndexTrend, signalSide, signalFreq string, longOnly bool) string {
	if it == nil || len(it.Timeframes) == 0 {
		return ""
	}
	var b strings.Builder
	if c := strings.TrimSpace(it.Code); c != "" {
		fmt.Fprintf(&b, "标的 %s\n", c)
	}
	if t := strings.TrimSpace(it.AnchorTime); t != "" {
		fmt.Fprintf(&b, "锚点 %s\n", t)
	}
	if v := strings.TrimSpace(it.StrategyVersion); v != "" {
		fmt.Fprintf(&b, "策略版本 %s\n", signalclientNormalizeAlpha(v))
	}
	sig := NormalizeFrequency(signalFreq)
	if sig == "" {
		sig = "60m"
	}
	const dailyKey = "daily"
	sigKey := priceanalysisLabFreq(sig)
	if tf, ok := it.Timeframes[dailyKey]; ok && tf.OK {
		appendLabIndexTrendTF(&b, "日线（方向层）", tf)
	}
	if sigKey != dailyKey {
		if tf, ok := it.Timeframes[sigKey]; ok && tf.OK {
			appendLabIndexTrendTF(&b, fmt.Sprintf("%s（信号周期）", labTATimeframeLabel(sigKey)), tf)
		}
	}
	for key, tf := range it.Timeframes {
		if !tf.OK {
			continue
		}
		if key == dailyKey || key == sigKey {
			continue
		}
		appendLabIndexTrendTF(&b, labTATimeframeLabel(key), tf)
	}
	if heading := LabSignalSideTriggerHeading(signalSide, longOnly); heading != "" {
		b.WriteString("\n")
		b.WriteString(heading)
		b.WriteByte('\n')
	}
	b.WriteString("\n规则提示：三旗=ADX趋势、EMA趋势(中长期)、SAR抛物线趋势；stance long/short/flat 对应多/空/平；日线与信号周期同向则倾向该方向。\n")
	return strings.TrimSpace(b.String())
}

// ResolveIndexTrendJEV calls JEV Choice once; never falls back to local flag rules.
func ResolveIndexTrendJEV(ctx context.Context, jev *JevClient, it *LabIndexTrend, signalSide, signalFreq string, longOnly bool) IndexTrendJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyIndexTrendJEV(model)
	if !labIndexTrendInformative(it) {
		if it != nil && strings.TrimSpace(it.Error) != "" {
			base.Error = it.Error
		} else {
			base.Error = "lab index-trend state empty"
		}
		return base
	}
	state := BuildIndexTrendJEVState(it, signalSide, signalFreq, longOnly)
	if state == "" {
		base.Error = "lab index-trend state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中多周期 ADX / EMA(中长期) / SAR 三旗 stance 与 summary（多/空/平/net），" +
		"判断该信号上下文的单指标趋势方向。只能从给定选项中选一项。" +
		"规则：①日线与信号周期同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅一方有信息则跟随有信息的一方；④均不足或确实无法判断时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, indexTrendJEVCriteria)
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
	base.RecommendedReason = labIndexTrendJEVReasonFromSummary(ctx, jev, choice, idx, base.Choices, confidence, state)
	return base
}

func emptyIndexTrendJEV(model string) IndexTrendJEV {
	return IndexTrendJEV{
		Source:  "jev",
		Model:   model,
		Choices: priceAnalysisJEVChoicesFromProbs(nil),
		OK:      false,
	}
}

func labIndexTrendJEVReasonFromSummary(ctx context.Context, jev *JevClient, choice string, idx int, choices []PriceAnalysisJEVChoice, confidence float64, facts string) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizeIndexTrendJEVReason(ctx, choice, idx, choices, confidence, facts)
	if err != nil {
		return labJEVReasonSummaryFailed(err.Error())
	}
	return reason
}
