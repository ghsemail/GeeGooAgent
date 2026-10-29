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
	Facts             string                   `json:"facts,omitempty"`
}

// SummarizePriceAnalysisJEVReason calls Decision chat to produce a human-readable reason
// from the original JEV choice output only (does not re-run direction inference).
func (j *JevClient) SummarizePriceAnalysisJEVReason(
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
	sys := `你是 GeeGoo 实验室价格分析助手。用户 JSON 含：facts（与 JEV Choice 同源的多周期价格结构摘要）、以及 JEV 方向判定（recommended_choice、choices 概率、confidence）。
请用 1～2 句简明中文向交易者解释该结论的主要依据，须基于 facts 中的结构要点（摆动、BOS/CHOCH、区间位置等）；可轻提置信度，但不得只复述概率数字；不得臆造 facts 中不存在的内容，也不得在 facts 非空时声称「未提供…」；不得改变或重推方向；不要输出 JSON、markdown 或列表；不要套用固定模板句。
措辞：向交易者说明时用「JEV 判定」或「模型」，不要出现 System One 等英文产品名。`
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

func labJEVReasonFromSummary(ctx context.Context, jev *JevClient, choice string, idx int, choices []PriceAnalysisJEVChoice, confidence float64, facts string) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizePriceAnalysisJEVReason(ctx, choice, idx, choices, confidence, facts)
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
