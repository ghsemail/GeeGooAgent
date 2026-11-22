package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type macroOverallReasonPool struct {
	PoolWorkflow  string `json:"pool_workflow"`
	Label         string `json:"label,omitempty"`
	JEVDirection  string `json:"jev_direction"`
	ReasonExcerpt string `json:"reason_excerpt,omitempty"`
}

type macroOverallReasonPayload struct {
	OrchestrationDate string                     `json:"orchestration_date"`
	PoolCount         int                        `json:"pool_count"`
	RecommendedChoice string                     `json:"recommended_choice"`
	RecommendedIndex  int                        `json:"recommended_index"`
	Confidence        float64                    `json:"confidence,omitempty"`
	Choices           []PriceAnalysisJEVChoice   `json:"choices"`
	Pools             []macroOverallReasonPool   `json:"pools"`
}

// SummarizeMacroOverallJEVReason produces macro overall Chinese reason from Choice + pool excerpts.
func (j *JevClient) SummarizeMacroOverallJEVReason(
	ctx context.Context,
	orchestrationDate, choice string,
	choiceIndex int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
	poolCount int,
	pools []macroOverallReasonPool,
) (string, error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", fmt.Errorf("jev not configured")
	}
	payload, err := json.Marshal(macroOverallReasonPayload{
		OrchestrationDate: strings.TrimSpace(orchestrationDate),
		PoolCount:         poolCount,
		RecommendedChoice: strings.TrimSpace(choice),
		RecommendedIndex:  choiceIndex,
		Confidence:        confidence,
		Choices:           choices,
		Pools:             pools,
	})
	if err != nil {
		return "", err
	}
	sys := `你是 GeeGoo 宏观整体风险偏好决策助手。用户 JSON 含：宏观整体 JEV 方向（recommended_choice、choices、confidence）、编排日、参与池数量 pool_count，以及 pools（各池 pool_workflow、jev_direction、reason_excerpt）。
当 pool_count 小于 7 时，须在正文中明示结论仅基于已参与池，勿声称覆盖全部资产类别。
请用简体中文 Markdown：以「## 整体结论」开头，给出风险偏好方向（看多=risk-on/看空=risk-off/不明确）、置信度、2-4 条核心依据；优先引用各池 reason_excerpt；禁止英文 chain-of-thought、禁止固定权重或多数票模板；不得改变或重推整体方向。
措辞：用「JEV 决策」或「模型」，不要出现 System One 等英文产品名。`
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
		return "", fmt.Errorf("empty overall reason")
	}
	return reason, nil
}