package report

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// IntradayHourlySummary is LLM-condensed hourly analysis for report bodies.
type IntradayHourlySummary struct {
	Price  string `json:"price_analysis"`
	Signal string `json:"signal_analysis"`
	Kline  string `json:"kline_analysis"`
}

// SummarizeIntradayHourly turns raw MCP hourly blocks into prose-only report sections.
func (s *Synthesizer) SummarizeIntradayHourly(
	ctx context.Context,
	ws memory.StockWorkspace,
	priceRaw, signalRaw, klineRaw string,
) (IntradayHourlySummary, error) {
	if !s.Available() {
		return IntradayHourlySummary{}, fmt.Errorf("synthesizer not available")
	}
	if strings.TrimSpace(priceRaw+signalRaw+klineRaw) == "" {
		return IntradayHourlySummary{}, fmt.Errorf("no hourly raw input")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	prompt := buildIntradayHourlySummaryPrompt(ws, priceRaw, signalRaw, klineRaw)
	callCtx := llm.WithCallMeta(cctx, llm.CallMeta{Kind: llm.TaskSynthesis})
	resp, err := s.gateway.Chat(callCtx, prompt, nil, "", 0)
	if err != nil {
		return IntradayHourlySummary{}, fmt.Errorf("intraday hourly summary LLM: %w", err)
	}
	parsed, err := parseIntradayHourlySummaryJSON(strings.TrimSpace(resp.Content))
	if err != nil {
		return IntradayHourlySummary{}, err
	}
	parsed.Price = sanitizeIntradayHourlySection(parsed.Price)
	parsed.Signal = sanitizeIntradayHourlySection(parsed.Signal)
	parsed.Kline = sanitizeIntradayHourlySection(parsed.Kline)
	if parsed.Price == "" && parsed.Signal == "" && parsed.Kline == "" {
		return IntradayHourlySummary{}, fmt.Errorf("intraday hourly summary empty")
	}
	return parsed, nil
}

func buildIntradayHourlySummaryPrompt(ws memory.StockWorkspace, priceRaw, signalRaw, klineRaw string) []llm.Message {
	var b strings.Builder
	b.WriteString("你是盘中报告编辑。将下列 MCP 小时级分析改写为面向用户的中文摘要。\n\n")
	b.WriteString(fmt.Sprintf("标的: %s (%s)\n\n", ws.StockName, ws.Code))
	writeHourlyRawBlock(&b, "价格分析原始稿", priceRaw)
	writeHourlyRawBlock(&b, "信号分析原始稿", signalRaw)
	writeHourlyRawBlock(&b, "K线分析原始稿", klineRaw)
	b.WriteString(`
要求:
- 仅输出 JSON: {"price_analysis":"...","signal_analysis":"...","kline_analysis":"..."}
- 某类原始稿为空时，对应字段输出空字符串 ""
- 禁止表格、禁止逐日 OHLC/成交量明细、禁止按日期逐行列举（如 8/3：482、07-29：6.15）
- 整体走势概述用 1 段话概括趋势、区间涨跌幅、关键形态，不要堆砌数字清单
- 量价/成交量用概括性描述（如「量能逐步萎缩」「下跌放量」），不要逐日罗列
- 综合判定/结论单独成段，观点明确
- 小节标题用 Markdown 加粗，如 **整体走势概述**、**量价关系**、**综合判定**
- 每条要点单独一行；段落之间空一行
- 不要 emoji、不要引用 [ev_...]、不要英文枚举`)
	return []llm.Message{
		{Role: llm.RoleSystem, Content: "你是严格的 JSON 报告编辑器, 只输出 JSON, 不输出任何其它文字。"},
		{Role: llm.RoleUser, Content: b.String()},
	}
}

func writeHourlyRawBlock(b *strings.Builder, title, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		b.WriteString(title + ": (无)\n\n")
		return
	}
	if len([]rune(raw)) > 6000 {
		raw = string([]rune(raw)[:6000]) + "…"
	}
	b.WriteString(title + ":\n")
	b.WriteString(raw)
	b.WriteString("\n\n")
}

func parseIntradayHourlySummaryJSON(content string) (IntradayHourlySummary, error) {
	content = strings.TrimSpace(content)
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```json")
		content = strings.TrimPrefix(content, "```")
		content = strings.TrimSuffix(content, "```")
		content = strings.TrimSpace(content)
	}
	var out IntradayHourlySummary
	if err := json.Unmarshal([]byte(content), &out); err != nil {
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			if err2 := json.Unmarshal([]byte(content[start:end+1]), &out); err2 == nil {
				return out, nil
			}
		}
		return out, fmt.Errorf("intraday hourly summary parse: %w", err)
	}
	return out, nil
}

func sanitizeIntradayHourlySection(text string) string {
	text = strings.TrimSpace(stockfmt.RepairIntradayLineBreaks(stockfmt.LocalizeDecisionTerms(text)))
	if text == "" {
		return ""
	}
	if strings.Contains(text, "|") && strings.Contains(text, "---") {
		return ""
	}
	return text
}
