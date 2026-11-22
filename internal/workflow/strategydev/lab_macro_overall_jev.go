package strategydev

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

// MacroOverallPoolInput is one frozen pool-tier row for macro overall JEV.
type MacroOverallPoolInput struct {
	PoolWorkflow string
	JEVDirection string // long | short | neutral
	JEVReason    string
}

// MacroOverallJEV is macro overall JEV outcome (assemble-only).
type MacroOverallJEV struct {
	OK                bool
	Model             string
	RecommendedChoice string
	Direction         LabJEVUnifiedDirection
	Confidence        float64
	Choices           []PriceAnalysisJEVChoice
	RecommendedReason string
	Error             string
}

var macroOverallPoolLabels = map[string]string{
	"cn_a":         "A股池",
	"hk":           "港股池",
	"us":           "美股池",
	"commodity_fx": "商品货币池",
	"bond":         "债券利率池",
	"crypto":       "虚拟货币池",
	"volatility":   "波动率池",
}

var macroOverallJEVCriteria = map[string]string{
	"看多":  "参与各池事实与池级 JEV 综合后全球风险偏好偏 risk-on；债券/波动/商品货币池方向不得机械等同于股指方向",
	"看空":  "参与各池事实与池级 JEV 综合后全球风险偏好偏 risk-off；须权衡各池理由而非固定权重计票",
	"不明确": "参与池信息不足以可靠判断宏观风险偏好，或冲突后仍无法权衡出明确 risk-on/off",
}

const maxPoolReasonInOverallState = 600

// BuildMacroOverallJEVState feeds overall Choice with per-pool conclusions.
func BuildMacroOverallJEVState(orchestrationDate string, pools []MacroOverallPoolInput) string {
	var b strings.Builder
	date := strings.TrimSpace(orchestrationDate)
	if date != "" {
		fmt.Fprintf(&b, "宏观编排日：%s\n", date)
	}
	n := len(pools)
	fmt.Fprintf(&b, "本次参与监控池数：%d", n)
	if n > 0 && n < 7 {
		b.WriteString("（非七池全量；结论仅覆盖下列池，勿声称全资产覆盖）")
	}
	b.WriteByte('\n')
	for _, p := range pools {
		wf := strings.TrimSpace(p.PoolWorkflow)
		if wf == "" {
			continue
		}
		label := macroOverallPoolLabels[wf]
		if label == "" {
			label = wf
		}
		choice := ledgerDirectionToChoice(p.JEVDirection)
		reason := strings.TrimSpace(p.JEVReason)
		if len(reason) > maxPoolReasonInOverallState {
			reason = reason[:maxPoolReasonInOverallState] + "…"
		}
		fmt.Fprintf(&b, "\n【%s (%s)】\n", label, wf)
		fmt.Fprintf(&b, "池级 JEV 方向（风险偏好语义）：%s\n", choice)
		if reason != "" {
			fmt.Fprintf(&b, "池级依据：%s\n", reason)
		}
	}
	return strings.TrimSpace(b.String())
}

// ResolveMacroOverallJEV runs Choice then Chinese overall reason (no conflict cap).
func ResolveMacroOverallJEV(
	ctx context.Context,
	jev *JevClient,
	orchestrationDate string,
	pools []MacroOverallPoolInput,
) MacroOverallJEV {
	model := config.DefaultDecisionModel
	if jev != nil && strings.TrimSpace(jev.cfg.Model) != "" {
		model = strings.TrimSpace(jev.cfg.Model)
	}
	base := MacroOverallJEV{Model: model, Choices: priceAnalysisJEVChoicesFromProbs(nil)}
	state := BuildMacroOverallJEVState(orchestrationDate, pools)
	if state == "" {
		base.Error = "overall decision state empty"
		return base
	}
	if jev == nil || strings.TrimSpace(jev.cfg.APIKey) == "" {
		base.Error = "jev not configured"
		return base
	}
	instructions := "基于 state 中各监控池的池级 JEV 方向与「池级依据」全文，综合判断当日全球宏观风险偏好（risk-on 对应看多、risk-off 对应看空）。" +
		"债券利率池、波动率池、商品货币池的方向含义与股指池不同，禁止机械等同或固定权重计票/多数票。" +
		"只能从给定选项中选一项。①多池互相印证 risk-on/off 时倾向对应方向；②冲突时须阅读依据权衡；③仅部分池有信息时跟随有依据的一方；" +
		"④不足时才选不明确。"
	choice, probs, confidence, respModel, err := jev.Choice(ctx, state, "direction", instructions, macroOverallJEVCriteria)
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
	base.Choices = priceAnalysisJEVChoicesFromProbs(probs)
	base.RecommendedChoice = choice
	conf := 0.0
	if idx >= 0 && idx < len(priceAnalysisJEVLabels) {
		if p, ok := probs[priceAnalysisJEVLabels[idx]]; ok {
			conf = clampUnitInterval(p)
		}
	}
	if confidence > 0 {
		conf = clampUnitInterval(confidence)
	}
	var reasonPools []macroOverallReasonPool
	for _, p := range pools {
		reasonPools = append(reasonPools, macroOverallReasonPool{
			PoolWorkflow:  strings.TrimSpace(p.PoolWorkflow),
			Label:         macroOverallPoolLabels[strings.TrimSpace(p.PoolWorkflow)],
			JEVDirection:  strings.TrimSpace(strings.ToLower(p.JEVDirection)),
			ReasonExcerpt: macroPoolReasonExcerpt(p.JEVReason),
		})
	}
	reason, rerr := jev.SummarizeMacroOverallJEVReason(ctx, orchestrationDate, choice, idx, base.Choices, conf, len(pools), reasonPools)
	if rerr != nil {
		base.Error = rerr.Error()
		return base
	}
	base.OK = true
	base.Error = ""
	base.Direction = choiceToLedgerDirection(choice)
	base.Confidence = conf
	base.RecommendedReason = reason
	return base
}
