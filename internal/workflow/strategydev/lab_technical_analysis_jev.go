package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

var technicalAnalysisJEVCriteria = map[string]string{
	"看多": "三重滤网：日线 Impulse/价值区/MACD/Force 整体偏多，信号周期同向或仅作顺势过滤",
	"看空": "三重滤网：日线偏空且信号周期确认或不过度逆势过滤",
	"不明确": "日线与信号周期严重冲突、Impulse 多为红区、或关键指标不足以可靠定向",
}

// BuildTechnicalAnalysisJEVState compresses multi-TF Elder indicators into state for JEV 决策.
func BuildTechnicalAnalysisJEVState(ta *LabTechnicalAnalysis, signalSide, signalFreq string) string {
	if ta == nil || len(ta.Timeframes) == 0 {
		return ""
	}
	var b strings.Builder
	if c := strings.TrimSpace(ta.Code); c != "" {
		fmt.Fprintf(&b, "标的 %s\n", c)
	}
	if t := strings.TrimSpace(ta.AnchorTime); t != "" {
		fmt.Fprintf(&b, "锚点 %s\n", t)
	}
	if v := strings.TrimSpace(ta.StrategyVersion); v != "" {
		fmt.Fprintf(&b, "策略版本 %s\n", signalclientNormalizeAlpha(v))
	}
	sig := NormalizeFrequency(signalFreq)
	if sig == "" {
		sig = "60m"
	}
	const dailyKey = "daily"
	sigKey := priceanalysisLabFreq(sig)
	if tf, ok := ta.Timeframes[dailyKey]; ok && tf.OK {
		appendLabTAIndicators(&b, "日线（三重滤网方向层）", tf.Indicators)
	}
	if sigKey != dailyKey {
		if tf, ok := ta.Timeframes[sigKey]; ok && tf.OK {
			appendLabTAIndicators(&b, fmt.Sprintf("%s（三重滤网过滤层）", labTATimeframeLabel(sigKey)), tf.Indicators)
		}
	}
	for key, tf := range ta.Timeframes {
		if !tf.OK {
			continue
		}
		if key == dailyKey || key == sigKey {
			continue
		}
		appendLabTAIndicators(&b, labTATimeframeLabel(key), tf.Indicators)
	}
	side := strings.TrimSpace(signalSide)
	if side == "buy" {
		b.WriteString("\n信号侧：买入触发\n")
	} else if side == "sell" {
		b.WriteString("\n信号侧：卖出触发\n")
	}
	b.WriteString("\n规则提示：日线定方向，信号周期过滤；价值区=EMA13与EMA26之间；Impulse 由 EMA13 斜率与 MACD 柱斜率同向绿/反向蓝/冲突红。\n")
	return strings.TrimSpace(b.String())
}

func signalclientNormalizeAlpha(v string) string {
	v = strings.TrimSpace(v)
	if v == "α" || strings.EqualFold(v, "a") {
		return "alpha"
	}
	return v
}

func priceanalysisLabFreq(sig string) string {
	switch NormalizeFrequency(sig) {
	case "5m":
		return "5m"
	case "60m":
		return "60m"
	case "daily":
		return "daily"
	case "weekly", "1wk":
		return "weekly"
	default:
		return NormalizeFrequency(sig)
	}
}

// ResolveTechnicalAnalysisJEV calls JEV Choice once; never falls back to local indicator rules.
func ResolveTechnicalAnalysisJEV(ctx context.Context, jev *JevClient, ta *LabTechnicalAnalysis, signalSide, signalFreq string) TechnicalAnalysisJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := emptyTechnicalAnalysisJEV(model)
	if !labTAInformative(ta) {
		if ta != nil && strings.TrimSpace(ta.Error) != "" {
			base.Error = ta.Error
		} else {
			base.Error = "lab technical-analysis state empty"
		}
		return base
	}
	state := BuildTechnicalAnalysisJEVState(ta, signalSide, signalFreq)
	if state == "" {
		base.Error = "lab technical-analysis state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中的 Elder 工具箱指标（价值区、Envelope、MACD 柱与 Impulse、Stochastic、RSI、Force Index），" +
		"按三重滤网（日线定方向、信号周期过滤）判断该信号上下文的技术面方向。只能从给定选项中选一项。" +
		"规则：①日线与过滤周期同向则选该方向；②冲突时须权衡后选看多或看空，不得仅因冲突选不明确；" +
		"③仅日线或仅过滤层一方有信息则跟随有信息的一方；④两方均不足或确实无法判断时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, technicalAnalysisJEVCriteria)
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
	base.RecommendedReason = labTAJEVReasonFromSummary(ctx, jev, choice, idx, base.Choices, confidence)
	return base
}

func emptyTechnicalAnalysisJEV(model string) TechnicalAnalysisJEV {
	return TechnicalAnalysisJEV{
		Source:  "jev",
		Model:   model,
		Choices: priceAnalysisJEVChoicesFromProbs(nil),
		OK:      false,
	}
}

func labTAJEVReasonFromSummary(ctx context.Context, jev *JevClient, choice string, idx int, choices []PriceAnalysisJEVChoice, confidence float64) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizeTechnicalAnalysisJEVReason(ctx, choice, idx, choices, confidence)
	if err != nil {
		return labJEVReasonSummaryFailed(err.Error())
	}
	return reason
}
