package strategydev

import (
	"context"
	"fmt"
	"strings"
)

// The cross-pool overview concludes one day's macro news. It reads the pool verdicts
// and a few item summaries, never the articles.

// MacroNewsOverviewPool is one pool verdict entering the overview.
type MacroNewsOverviewPool struct {
	PoolID    string
	Label     string
	Direction string // long | short | neutral
	Reason    string

	Bullish, Bearish, Neutral, HighValueCount int

	Items []MacroNewsPoolItem
}

const (
	maxOverviewReasonRunes     = 600
	maxOverviewItemReasonRunes = 120
	maxOverviewItems           = 8
)

// A volatility verdict is about volatility itself, so it reads the other way round.
const macroNewsOverviewScale = "整体方向指跨市场风险偏好：看多=偏向风险偏好（risk-on），看空=偏向避险（risk-off）。" +
	"各池方向是对该池标的而言；波动率池的看多表示波动率上行，通常对应避险；债券与商品货币池须结合理由判断其对风险偏好的含义。"

const macroNewsOverviewSummarySystem = `你是 GeeGoo 跨市场宏观新闻综合决策助手。用户 JSON 含当日宏观新闻的整体 JEV 方向判定（recommended_choice、choices、confidence、conflict）、pools（已有当日结论的监控池：方向、结论、多空中性条数、高价值条数与少量代表条目）与 missing_pools（当日没有结论的池）。
` + macroNewsOverviewScale + `
` + macroNewsDirectionRule + `
请用简体中文 Markdown 写当日跨市场的宏观新闻结论：以「## 整体结论」开头，说明整体方向（用看多/看空/不明确表述）与置信度；随后列 2-4 条核心依据，写跨池的共同主线与传导关系，不要逐池复述；若各池存在分歧，用一句话说明主要风险；missing_pools 不为空时，最后用一句话注明结论未覆盖这些池。总长不超过 600 字。只返回结论正文，不含英文推理过程。
` + MacroNewsJEVPrinciples

// BuildMacroNewsOverviewState builds the Decision Choice state of the overview.
func BuildMacroNewsOverviewState(date string, pools []MacroNewsOverviewPool, missing []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "日期：%s\n%s\n", date, macroNewsOverviewScale)
	if len(missing) > 0 {
		fmt.Fprintf(&b, "当日没有结论的池：%s\n", strings.Join(missing, "、"))
	}
	for _, pool := range pools {
		fmt.Fprintf(&b, "\n【%s】\n池级 JEV 方向：%s\n高价值新闻 %d 条（看多 %d，看空 %d，中性 %d）\n",
			pool.Label, ledgerDirectionToChoice(pool.Direction), pool.HighValueCount, pool.Bullish, pool.Bearish, pool.Neutral)
		if reason := strings.TrimSpace(pool.Reason); reason != "" {
			fmt.Fprintf(&b, "池结论：%s\n", truncateRunes(reason, maxOverviewReasonRunes))
		}
		for _, item := range pool.Items[:min(len(pool.Items), maxOverviewItems)] {
			fmt.Fprintf(&b, "- %s（%s）%s\n", truncateRunes(item.Title, 120),
				ledgerDirectionToChoice(item.JEVDirection), truncateRunes(item.JEVReason, maxOverviewItemReasonRunes))
		}
	}
	return strings.TrimSpace(b.String())
}

type macroNewsOverviewPoolPayload struct {
	Pool           string                    `json:"pool"`
	JEVDirection   string                    `json:"jev_direction"`
	Conclusion     string                    `json:"conclusion,omitempty"`
	HighValueCount int                       `json:"high_value_count"`
	Bullish        int                       `json:"bullish"`
	Bearish        int                       `json:"bearish"`
	Neutral        int                       `json:"neutral"`
	Headlines      []macroNewsReasonHeadline `json:"headlines"`
}

type macroNewsOverviewPayload struct {
	Date              string                         `json:"date"`
	RecommendedChoice string                         `json:"recommended_choice"`
	Confidence        float64                        `json:"confidence,omitempty"`
	Conflict          bool                           `json:"conflict,omitempty"`
	Choices           []PriceAnalysisJEVChoice       `json:"choices"`
	Pools             []macroNewsOverviewPoolPayload `json:"pools"`
	MissingPools      []string                       `json:"missing_pools"`
}

// ResolveMacroNewsOverviewJEV gets the overall direction from Decision Choice (never
// by counting pools) and has the synthesis model write the conclusion. missing holds
// the labels of pools without a verdict.
func ResolveMacroNewsOverviewJEV(
	ctx context.Context,
	jev *JevClient,
	date string,
	pools []MacroNewsOverviewPool,
	missing []string,
	summarizer MacroNewsSummarizer,
) MacroNewsVerdict {
	if len(pools) == 0 {
		return MacroNewsVerdict{MacroPoolJEV: MacroPoolJEV{
			Choices: priceAnalysisJEVChoicesFromProbs(nil), Error: "no pool verdict",
		}}
	}
	hasLong, hasShort := false, false
	for _, pool := range pools {
		direction := pool.Direction
		if pool.PoolID == "volatility" {
			// Rising volatility argues against risk appetite.
			switch direction {
			case "long":
				direction = "short"
			case "short":
				direction = "long"
			}
		}
		hasLong = hasLong || direction == "long"
		hasShort = hasShort || direction == "short"
	}
	instructions := "基于 state 中各监控池当日结论与代表新闻，综合判断当日宏观新闻的整体方向。" +
		macroNewsOverviewScale + macroNewsChoiceRules
	out := chooseMacroNewsDirection(ctx, jev, BuildMacroNewsOverviewState(date, pools, missing), instructions, hasLong, hasShort)
	if !out.OK {
		return out
	}
	payload := macroNewsOverviewPayload{
		Date: date, RecommendedChoice: out.RecommendedChoice, Confidence: out.Confidence,
		Conflict: out.Conflict, Choices: out.Choices, MissingPools: append([]string{}, missing...),
	}
	for _, pool := range pools {
		entry := macroNewsOverviewPoolPayload{
			Pool: pool.Label, JEVDirection: ledgerDirectionToChoice(pool.Direction),
			Conclusion: truncateRunes(pool.Reason, maxOverviewReasonRunes), HighValueCount: pool.HighValueCount,
			Bullish: pool.Bullish, Bearish: pool.Bearish, Neutral: pool.Neutral,
			Headlines: []macroNewsReasonHeadline{},
		}
		for _, item := range pool.Items[:min(len(pool.Items), maxOverviewItems)] {
			entry.Headlines = append(entry.Headlines, macroNewsReasonHeadline{
				Title: truncateRunes(item.Title, 120), JEVDirection: item.JEVDirection,
				Reason: truncateRunes(item.JEVReason, maxOverviewItemReasonRunes),
			})
		}
		payload.Pools = append(payload.Pools, entry)
	}
	summary, err := writeMacroNewsSummary(ctx, summarizer, macroNewsOverviewSummarySystem, payload)
	out.setReason(summary, err)
	return out
}
