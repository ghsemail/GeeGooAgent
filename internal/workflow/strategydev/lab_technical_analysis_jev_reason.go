package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SummarizeTechnicalAnalysisJEVReason calls Decision chat for TA JEV 决策 reason text.
func (j *JevClient) SummarizeTechnicalAnalysisJEVReason(
	ctx context.Context,
	choice string,
	choiceIndex int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
	facts string,
) (string, error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", fmt.Errorf("jev not configured")
	}
	payload, err := json.Marshal(priceAnalysisJEVReasonPayload{
		RecommendedChoice: strings.TrimSpace(choice),
		RecommendedIndex:  choiceIndex,
		Confidence:        confidence,
		Choices:           choices,
		Facts:             strings.TrimSpace(facts),
	})
	if err != nil {
		return "", err
	}
	sys := `你是 GeeGoo 实验室技术分析助手。用户 JSON 含：facts（Elder 指标与三重滤网摘要，与 JEV Choice 同源）、以及 JEV 方向判定（recommended_choice、choices 概率、confidence）。
请用 1～2 句简明中文向交易者解释该结论的主要依据，须引用 facts 中的指标读数（价值区、Impulse、MACD、Stochastic、RSI、Force Index、三重滤网等）；可轻提置信度，但不得只复述概率数字；不得臆造 facts 中不存在的内容，也不得在 facts 非空时声称「未提供指标」或「未提供…」；不得改变或重推方向；不要输出 JSON、markdown 或列表。
措辞：向交易者说明时用「JEV 决策」或「模型」，不要出现 System One 等英文产品名。`
	var content string
	if j.reasonChat != nil {
		content, err = j.reasonChat(ctx, sys, string(payload))
	} else {
		content, err = j.decisionChat(ctx, sys, string(payload))
	}
	if err != nil {
		return "", err
	}
	reason := sanitizeLabJEVReasonSummary(content)
	if reason == "" {
		return "", fmt.Errorf("empty reason summary")
	}
	return reason, nil
}
