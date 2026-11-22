package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type macroPoolReasonSymbol struct {
	Symbol       string  `json:"symbol"`
	Direction    string  `json:"jev_direction"`
	Confidence   float64 `json:"confidence,omitempty"`
	ReasonExcerpt string `json:"reason_excerpt,omitempty"`
}

type macroPoolReasonPayload struct {
	PoolWorkflow      string                  `json:"pool_workflow"`
	Title             string                  `json:"title,omitempty"`
	RecommendedChoice string                  `json:"recommended_choice"`
	RecommendedIndex  int                     `json:"recommended_index"`
	Confidence        float64                 `json:"confidence,omitempty"`
	Conflict          bool                    `json:"conflict,omitempty"`
	Choices           []PriceAnalysisJEVChoice `json:"choices"`
	Symbols           []macroPoolReasonSymbol `json:"symbols"`
}

// SummarizeMacroPoolJEVReason produces pool-level Chinese reason from Choice + symbol excerpts.
func (j *JevClient) SummarizeMacroPoolJEVReason(
	ctx context.Context,
	poolWorkflow, title, choice string,
	choiceIndex int,
	choices []PriceAnalysisJEVChoice,
	confidence float64,
	conflict bool,
	symbols []macroPoolReasonSymbol,
) (string, error) {
	if j == nil || strings.TrimSpace(j.cfg.APIKey) == "" {
		return "", fmt.Errorf("jev not configured")
	}
	payload, err := json.Marshal(macroPoolReasonPayload{
		PoolWorkflow:      strings.TrimSpace(poolWorkflow),
		Title:             strings.TrimSpace(title),
		RecommendedChoice: strings.TrimSpace(choice),
		RecommendedIndex:  choiceIndex,
		Confidence:        confidence,
		Conflict:          conflict,
		Choices:           choices,
		Symbols:           symbols,
	})
	if err != nil {
		return "", err
	}
	sys := `你是 GeeGoo 宏观监控池综合决策助手。用户 JSON 含：池级 JEV 方向判定（recommended_choice、choices、confidence、conflict），以及 symbols（各标的 symbol、jev_direction、reason_excerpt、confidence）。
请用简体中文 Markdown 写池级结论：以「## 整体结论」开头，给出方向（用看多/看空/不明确表述）、置信度、2-4 条核心依据；优先引用各标的 reason_excerpt，不得简单计数看多/看空票数或固定权重规则；不得改变或重推池级方向；正文禁止英文 chain-of-thought。
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
	reason := strings.TrimSpace(content)
	if reason == "" {
		return "", fmt.Errorf("empty pool reason")
	}
	return reason, nil
}