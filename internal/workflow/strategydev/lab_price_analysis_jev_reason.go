package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const labJEVReasonSummaryFailedPrefix = "理由总结失败："

type priceAnalysisJEVReasonPayload struct {
	RecommendedChoice string                   `json:"recommended_choice"`
	RecommendedIndex  int                      `json:"recommended_index"`
	Confidence        float64                  `json:"confidence,omitempty"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
}

// SummarizePriceAnalysisJEVReason calls Decision chat to produce a human-readable reason
// from the original JEV choice output only (does not re-run direction inference).
func (j *JevClient) SummarizePriceAnalysisJEVReason(
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
	sys := `你是 GeeGoo 实验室价格分析助手。用户会提供 System One（JEV）对「看多 / 看空 / 不明确」的原始输出（推荐项、各选项概率、置信度）。
请用 1～2 句简明中文向交易者解释该结论的主要依据；不得改变或重推方向；不要输出 JSON、markdown 或列表；不要套用固定模板句（例如「结构+形态 JEV 倾向…」）。`
	var content string
	if j.reasonChat != nil {
		content, err = j.reasonChat(ctx, sys, string(payload))
	} else {
		content, err = j.decisionChat(ctx, sys, string(payload))
	}
	if err != nil {
		return "", err
	}
	reason := strings.TrimSpace(content)
	if reason == "" {
		return "", fmt.Errorf("empty reason summary")
	}
	return reason, nil
}

func labJEVReasonFromSummary(ctx context.Context, jev *JevClient, choice string, idx int, choices []PriceAnalysisJEVChoice, confidence float64) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizePriceAnalysisJEVReason(ctx, choice, idx, choices, confidence)
	if err != nil {
		return labJEVReasonSummaryFailed(err.Error())
	}
	return reason
}

func labJEVReasonSummaryFailed(detail string) string {
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return labJEVReasonSummaryFailedPrefix + "未知错误"
	}
	return labJEVReasonSummaryFailedPrefix + detail
}
