package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// SummarizeKlineJEVReason calls Decision chat for K-line JEV 决策 reason text.
func (j *JevClient) SummarizeKlineJEVReason(
	ctx context.Context,
	choice string,
	choiceIndex int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
) (string, error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", fmt.Errorf("jev not configured")
	}
	payload, err := json.Marshal(priceAnalysisJEVReasonPayload{
		RecommendedChoice: strings.TrimSpace(choice),
		RecommendedIndex:  choiceIndex,
		Confidence:        confidence,
		Choices:           choices,
	})
	if err != nil {
		return "", err
	}
	sys := `你是 GeeGoo 实验室 K 线形态助手。用户会提供 JEV 方向判定对「看多 / 看空 / 不明确」的原始输出（推荐项、各选项概率、置信度）。
请用 1～2 句简明中文向交易者解释该结论的主要依据；不得改变或重推方向；不要输出 JSON、markdown 或列表。
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
