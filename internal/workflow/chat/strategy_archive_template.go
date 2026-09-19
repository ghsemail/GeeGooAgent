package chat

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/templates"
)

var (
	htmlCommentRE        = regexp.MustCompile(`(?s)<!--.*?-->`)
	sectionPlaceholderRE = regexp.MustCompile(`\{\{section:([^}]+)\}\}`)
)

func loadStrategyArchiveTemplate() string {
	return templates.LoadStrategyArchiveTemplate()
}

func cognitionComposeSystemPrompt() string {
	headings := archiveRequiredHeadings(loadStrategyArchiveTemplate())
	if len(headings) == 0 {
		headings = defaultArchiveHeadings()
	}
	var list strings.Builder
	for _, h := range headings {
		list.WriteString("   ## ")
		list.WriteString(h)
		list.WriteString("\n")
	}
	return `你是 GeeGoo 量化平台的策略档案文档写手。你会收到一份策略库 JSON（权威、可执行定义）。
请结合你对技术指标与量化策略的一般知识，写一份供 Agent 注入上下文使用的 Markdown 正文。

硬性要求：
1. 策略库 JSON 是执行真相；不得与其 buy_signal / sell_signal / 参数矛盾
2. 若 JSON 含 signal / flag / nosignal 等 type，必须解释在本策略中的组合含义（AND/OR、主触发 vs 过滤）
3. 必须写清：何时适合使用、何时不适合、关键参数及调节影响
4. 不得编造策略库里没有的额外入场/出场规则
5. 输出纯 Markdown 正文（不要用代码围栏包裹全文），必须包含以下章节（标题逐字一致）：
` + list.String() + `6. 「Agent 使用指引」需说明：用户问什么时应引用本策略、probe/回测前要确认哪些参数与周期
7. 用中文，简洁可检索，避免空话与重复 JSON 原文`
}

func defaultArchiveHeadings() []string {
	return []string{
		"一句话定位",
		"适用场景",
		"不适用 / 风险",
		"参数说明",
		"指标与信号逻辑",
		"买卖规则（与策略库对齐）",
		"周期与频率",
		"Agent 使用指引",
	}
}

func archiveRequiredHeadings(tpl string) []string {
	tpl = stripHTMLComments(tpl)
	var out []string
	for _, line := range strings.Split(tpl, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "## ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(line, "## "))
		if heading == "" || strings.HasPrefix(heading, "附录") {
			continue
		}
		out = append(out, heading)
	}
	return out
}

func assembleAgentCognitionDoc(flow *Flow, synthesizedBody string) string {
	tpl := loadStrategyArchiveTemplate()
	if strings.TrimSpace(tpl) == "" {
		return assembleArchiveDocFallback(flow, synthesizedBody)
	}
	return applyArchiveTemplate(tpl, flow, synthesizedBody)
}

func applyArchiveTemplate(tpl string, flow *Flow, synthesizedBody string) string {
	label := cognitionLabel(flow)
	now := time.Now().UTC().Format(time.RFC3339)
	signalID := catalogField(flow.CatalogRaw, "signal_id", "id")
	out := stripHTMLComments(tpl)
	out = strings.ReplaceAll(out, "{{strategy_name}}", label)
	out = strings.ReplaceAll(out, "{{catalog_type}}", strings.TrimSpace(flow.CatalogType))
	out = strings.ReplaceAll(out, "{{signal_id}}", signalID)
	out = strings.ReplaceAll(out, "{{generated_at}}", now)
	out = strings.ReplaceAll(out, "{{catalog_json}}", catalogJSON(flow.CatalogRaw))

	body := strings.TrimSpace(synthesizedBody)
	sections := splitMarkdownSections(body)
	out = sectionPlaceholderRE.ReplaceAllStringFunc(out, func(match string) string {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(match, "{{section:"), "}}"))
		if text, ok := sections[name]; ok {
			return text
		}
		return ""
	})
	if strings.Contains(out, "{{body}}") {
		out = strings.ReplaceAll(out, "{{body}}", body)
	}
	return strings.TrimSpace(out)
}

func assembleArchiveDocFallback(flow *Flow, synthesizedBody string) string {
	label := cognitionLabel(flow)
	now := time.Now().UTC().Format(time.RFC3339)
	signalID := catalogField(flow.CatalogRaw, "signal_id", "id")
	var b strings.Builder
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "doc_type: strategy_agent_archive\n")
	fmt.Fprintf(&b, "strategy_name: %s\n", label)
	fmt.Fprintf(&b, "catalog_type: %s\n", flow.CatalogType)
	if signalID != "" {
		fmt.Fprintf(&b, "signal_id: %s\n", signalID)
	}
	fmt.Fprintf(&b, "generated_at: %s\n", now)
	fmt.Fprintf(&b, "source: geegoo_catalog + llm\n")
	fmt.Fprintf(&b, "agent_use: inject when user asks about this strategy, before probe/backtest\n")
	fmt.Fprintf(&b, "---\n\n")
	fmt.Fprintf(&b, "# %s · 策略档案\n\n", label)
	fmt.Fprintf(&b, "> 供 Agent 注入上下文；**执行信号以策略库为准**。\n\n")
	b.WriteString(strings.TrimSpace(synthesizedBody))
	fmt.Fprintf(&b, "\n\n---\n\n## 附录 · 策略库原文\n\n")
	fmt.Fprintf(&b, "```json\n%s\n```\n", catalogJSON(flow.CatalogRaw))
	return strings.TrimSpace(b.String())
}

func splitMarkdownSections(md string) map[string]string {
	out := map[string]string{}
	var cur string
	var buf strings.Builder
	flush := func() {
		if cur != "" {
			out[cur] = strings.TrimSpace(buf.String())
		}
		buf.Reset()
	}
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "## ") {
			flush()
			cur = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "## "))
			continue
		}
		if cur != "" {
			buf.WriteString(line)
			buf.WriteByte('\n')
		}
	}
	flush()
	return out
}

func stripHTMLComments(s string) string {
	return strings.TrimSpace(htmlCommentRE.ReplaceAllString(s, ""))
}
