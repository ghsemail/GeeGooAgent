package strategydev

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
)

// The pool reason of a macro-news pool is a summary written by the synthesis model,
// like diagnose_macro_pool's. Decision Choice has already fixed the direction.

// MacroNewsSummarizer runs one synthesis prompt. accept rejects an unusable reply.
type MacroNewsSummarizer interface {
	Complete(ctx context.Context, system, user string, accept func(content string) error) (model string, err error)
}

const (
	// A pool can hold hundreds of items; the summary reads the most relevant ones,
	// with room kept for the other side.
	maxMacroNewsSummaryAligned = 16
	maxMacroNewsSummaryOther   = 8
	maxMacroNewsSummaryRunes   = 2000
	minMacroNewsSummaryHan     = 20

	macroNewsSummaryHeading = "## 整体结论"
)

var (
	errMacroNewsSummaryUnusable = errors.New("summary reply is unusable")
	errMacroNewsNoSummarizer    = errors.New("synthesis model not configured")
)

// macroNewsSummaryFailure is the stored reason when no summary could be written. It
// names the kind of failure only; provider errors are not stored.
func macroNewsSummaryFailure(err error) string {
	cause := "综合模型调用失败或没有给出可用结论"
	switch {
	case errors.Is(err, errMacroNewsNoSummarizer):
		cause = "综合模型未配置"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		cause = "超出本次调用的时间预算"
	}
	return "理由总结失败：" + cause + "。方向仍为本次 JEV 决策结果，下次运行会重试总结。"
}

const macroNewsPoolSummarySystem = `你是 GeeGoo 宏观监控池新闻综合决策助手。用户 JSON 含池级 JEV 方向判定（recommended_choice、choices、confidence、conflict），以及 headlines（从本池 news_total 条当日高价值新闻中选取的代表性条目：标题、jev_direction、理由）。
` + macroNewsDirectionRule + `
请用简体中文 Markdown 写本池的全池结论：以「## 整体结论」开头，说明方向（用看多/看空/不明确表述）与置信度；随后列 2-4 条核心依据，把同类事件归纳成一条，说明它为何影响本池，可引用代表性标题作佐证；若存在反向或分歧信息，用一句话说明主要风险。总长不超过 400 字。只返回结论正文，不含英文推理过程。
` + MacroNewsJEVPrinciples

type macroNewsSummaryPayload struct {
	PoolID            string                    `json:"pool_id"`
	PoolLabel         string                    `json:"pool_label,omitempty"`
	RecommendedChoice string                    `json:"recommended_choice"`
	Confidence        float64                   `json:"confidence,omitempty"`
	Conflict          bool                      `json:"conflict,omitempty"`
	Choices           []PriceAnalysisJEVChoice  `json:"choices"`
	NewsTotal         int                       `json:"news_total"`
	Headlines         []macroNewsReasonHeadline `json:"headlines"`
}

// macroNewsSummaryItems takes items that support the choice first, then others, so
// the summary can name what argues the other way.
func macroNewsSummaryItems(choice string, items []MacroNewsPoolItem) []MacroNewsPoolItem {
	var aligned, other []MacroNewsPoolItem
	for _, item := range items {
		c := ledgerDirectionToChoice(item.JEVDirection)
		if c == choice || (choice == "不明确" && c != "不明确") {
			aligned = append(aligned, item)
		} else {
			other = append(other, item)
		}
	}
	// Directional counter-evidence is more informative than neutral items.
	var opposing, neutral []MacroNewsPoolItem
	for _, item := range other {
		if item.JEVDirection == "long" || item.JEVDirection == "short" {
			opposing = append(opposing, item)
		} else {
			neutral = append(neutral, item)
		}
	}
	other = append(opposing, neutral...)
	if len(aligned) > maxMacroNewsSummaryAligned {
		aligned = aligned[:maxMacroNewsSummaryAligned]
	}
	if len(other) > maxMacroNewsSummaryOther {
		other = other[:maxMacroNewsSummaryOther]
	}
	return append(aligned, other...)
}

// cleanMacroNewsSummary strips reasoning blocks and fences. It returns "" for a
// reply that is not a Chinese summary.
func cleanMacroNewsSummary(raw string) string {
	text := strings.TrimSpace(raw)
	if i := strings.LastIndex(text, "</think>"); i >= 0 {
		text = strings.TrimSpace(text[i+len("</think>"):])
	}
	if strings.HasPrefix(text, "```") {
		if i := strings.Index(text, "\n"); i >= 0 {
			text = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text[i+1:]), "```"))
		}
	}
	if strings.HasPrefix(text, "{") || strings.HasPrefix(text, "[") {
		return ""
	}
	han := 0
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			han++
		}
	}
	if han < minMacroNewsSummaryHan {
		return ""
	}
	if !strings.HasPrefix(text, macroNewsSummaryHeading) {
		text = macroNewsSummaryHeading + "\n\n" + text
	}
	return truncateRunes(text, maxMacroNewsSummaryRunes)
}

// summarizeMacroNewsPool asks the synthesis model for the pool conclusion.
func summarizeMacroNewsPool(
	ctx context.Context,
	summarizer MacroNewsSummarizer,
	poolID, poolLabel string,
	verdict MacroPoolJEV,
	items []MacroNewsPoolItem,
) (string, error) {
	payload := macroNewsSummaryPayload{
		PoolID: poolID, PoolLabel: poolLabel, RecommendedChoice: verdict.RecommendedChoice,
		Confidence: verdict.Confidence, Conflict: verdict.Conflict, Choices: verdict.Choices,
		NewsTotal: len(items),
	}
	for _, item := range macroNewsSummaryItems(verdict.RecommendedChoice, items) {
		payload.Headlines = append(payload.Headlines, macroNewsReasonHeadline{
			Title: truncateRunes(item.Title, 160), JEVDirection: item.JEVDirection,
			Reason: truncateRunes(item.JEVReason, 160),
		})
	}
	return writeMacroNewsSummary(ctx, summarizer, macroNewsPoolSummarySystem, payload)
}

// writeMacroNewsSummary runs one summary prompt and returns the cleaned Markdown.
func writeMacroNewsSummary(ctx context.Context, summarizer MacroNewsSummarizer, system string, payload any) (string, error) {
	if summarizer == nil {
		return "", errMacroNewsNoSummarizer
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	summary := ""
	if _, err = summarizer.Complete(ctx, system, string(raw), func(content string) error {
		cleaned := cleanMacroNewsSummary(content)
		if cleaned == "" {
			return errMacroNewsSummaryUnusable
		}
		summary = cleaned
		return nil
	}); err != nil {
		return "", err
	}
	if summary == "" {
		return "", errMacroNewsSummaryUnusable
	}
	return summary, nil
}
