package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type labJEVUnifiedLegReasonInput struct {
	Name              string `json:"name"`
	RecommendedChoice string `json:"recommended_choice,omitempty"`
	RecommendedReason string `json:"recommended_reason,omitempty"`
	Participated      bool   `json:"participated"`
}

type labJEVUnifiedReasonPayload struct {
	RecommendedChoice string                        `json:"recommended_choice"`
	RecommendedIndex  int                           `json:"recommended_index"`
	Confidence        float64                       `json:"confidence,omitempty"`
	Choices           []PriceAnalysisJEVChoice      `json:"choices"`
	LegConclusions    []labJEVUnifiedLegReasonInput `json:"leg_conclusions"`
}

// SummarizeLabJEVUnifiedReason calls Decision chat for unified JEV 决策 reason text.
func (j *JevClient) SummarizeLabJEVUnifiedReason(
	ctx context.Context,
	choice string,
	choiceIndex int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
	legInputs []labJEVUnifiedLegReasonInput,
) (string, error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", fmt.Errorf("jev not configured")
	}
	payload, err := json.Marshal(labJEVUnifiedReasonPayload{
		RecommendedChoice: strings.TrimSpace(choice),
		RecommendedIndex:  choiceIndex,
		Confidence:        confidence,
		Choices:           choices,
		LegConclusions:    legInputs,
	})
	if err != nil {
		return "", err
	}
	sys := `你是 GeeGoo 实验室综合决策助手。用户会提供总 JEV 方向判定（推荐项、各选项概率、置信度），以及参与合成的各路 JEV 结论摘要（价格结构 / K 线形态 / 技术指标；未参与的路会标明 participated=false）。
请用 1～3 句简明中文向交易者解释总方向的主要依据；须引用已参与各路结论中的要点，不得声称「未提供指标」；不得改变或重推总方向；不要输出 JSON、markdown 或列表；不要用固定权重模板句。
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

func labJEVUnifiedReasonFromSummary(
	ctx context.Context,
	jev *JevClient,
	choice string,
	idx int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
	legInputs []labJEVUnifiedLegReasonInput,
) string {
	if jev == nil {
		return labJEVReasonSummaryFailed("jev not configured")
	}
	reason, err := jev.SummarizeLabJEVUnifiedReason(ctx, choice, idx, choices, confidence, legInputs)
	if err != nil {
		return labJEVReasonSummaryFailed(err.Error())
	}
	return reason
}
