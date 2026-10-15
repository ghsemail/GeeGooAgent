package strategydev

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// InterceptConclusionLLM synthesizes one-line conclusions for triggers (batch).
type InterceptConclusionLLM interface {
	Chat(ctx context.Context, messages []llm.Message, tools []llm.ToolSchema, temperature float64, maxTokens int) (*llm.Response, error)
}

// EnrichEpisodeStatsAIConclusions replaces rule conclusions with LLM text when available.
func EnrichEpisodeStatsAIConclusions(ctx context.Context, llmClient InterceptConclusionLLM, stats *EpisodeTriggerFilterStats) {
	if stats == nil || llmClient == nil {
		return
	}
	type item struct {
		Key            string `json:"key"`
		Side           string `json:"side"`
		Time           string `json:"time"`
		Masked         bool   `json:"masked"`
		WouldIntercept bool   `json:"would_intercept"`
		Rule           string `json:"rule_conclusion"`
		Summary        string `json:"price_summary,omitempty"`
	}
	var batch []item
	index := map[string]*EpisodeTriggerRecord{}
	for i := range stats.BuyTriggers {
		tr := &stats.BuyTriggers[i]
		key := fmt.Sprintf("buy:%d", tr.Index)
		index[key] = tr
		sum := ""
		if tr.PriceAnalysis != nil {
			sum = tr.PriceAnalysis.Summary
		}
		batch = append(batch, item{Key: key, Side: "buy", Time: tr.Time, Masked: tr.Masked, WouldIntercept: tr.WouldIntercept, Rule: tr.AIInterceptConclusion, Summary: sum})
	}
	for i := range stats.SellTriggers {
		tr := &stats.SellTriggers[i]
		key := fmt.Sprintf("sell:%d", tr.Index)
		index[key] = tr
		sum := ""
		if tr.PriceAnalysis != nil {
			sum = tr.PriceAnalysis.Summary
		}
		batch = append(batch, item{Key: key, Side: "sell", Time: tr.Time, Masked: tr.Masked, WouldIntercept: tr.WouldIntercept, Rule: tr.AIInterceptConclusion, Summary: sum})
	}
	if len(batch) == 0 {
		return
	}
	raw, err := json.Marshal(batch)
	if err != nil {
		return
	}
	user := fmt.Sprintf(`以下为同一次信号诊断中各触发的多周期价格分析结论草稿（rule 字段）。请为每条写一句中文 ai_intercept_conclusion。
规则：would_intercept=false 且 masked=false 时必须写「保留」或「通过」，不得写「已拦截」；would_intercept=true 且 masked=false 写「本会拦截」；masked=true 写「已拦截」。锚点不可用不等于已掩码拦截。
返回 JSON 数组：[{"key":"buy:3","conclusion":"..."}]，key 必须与输入一致。

输入：
%s`, string(raw))
	resp, err := llmClient.Chat(ctx, []llm.Message{
		{Role: llm.RoleSystem, Content: "你是 GeeGoo 价格结构拦截助手。只输出 JSON 数组，不要 markdown。"},
		{Role: llm.RoleUser, Content: user},
	}, nil, 0, 512)
	if err != nil {
		return
	}
	text := strings.TrimSpace(llm.VisibleAssistantContent(resp.Content, resp.ReasoningContent))
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	var out []struct {
		Key        string `json:"key"`
		Conclusion string `json:"conclusion"`
	}
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return
	}
	for _, row := range out {
		tr, ok := index[row.Key]
		if !ok || strings.TrimSpace(row.Conclusion) == "" {
			continue
		}
		tr.AIInterceptConclusion = strings.TrimSpace(row.Conclusion)
		tr.AIInterceptConclusionSource = "llm"
	}
	syncStrippedKeptFromTriggers(stats)
}

func syncStrippedKeptFromTriggers(stats *EpisodeTriggerFilterStats) {
	stats.StrippedBuyTriggers = nil
	stats.StrippedSellTriggers = nil
	stats.KeptBuyTriggers = nil
	stats.KeptSellTriggers = nil
	for _, tr := range stats.BuyTriggers {
		if tr.Masked {
			stats.StrippedBuyTriggers = append(stats.StrippedBuyTriggers, tr)
		} else {
			stats.KeptBuyTriggers = append(stats.KeptBuyTriggers, tr)
		}
	}
	for _, tr := range stats.SellTriggers {
		if tr.Masked {
			stats.StrippedSellTriggers = append(stats.StrippedSellTriggers, tr)
		} else {
			stats.KeptSellTriggers = append(stats.KeptSellTriggers, tr)
		}
	}
}
