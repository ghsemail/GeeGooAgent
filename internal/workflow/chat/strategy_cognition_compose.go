package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const cognitionComposeSystemPrompt = `你是 GeeGoo 量化平台的策略认知文档写手。你会收到一份策略库 JSON（权威、可执行定义）。
请结合你对技术指标与量化策略的一般知识，写一份供 Agent 注入上下文使用的 Markdown 正文。

硬性要求：
1. 策略库 JSON 是执行真相；不得与其 buy_signal / sell_signal / 参数矛盾
2. 若 JSON 含 signal / flag / nosignal 等 type，必须解释在本策略中的组合含义（AND/OR、主触发 vs 过滤）
3. 必须写清：何时适合使用、何时不适合、关键参数及调节影响
4. 不得编造策略库里没有的额外入场/出场规则
5. 输出纯 Markdown 正文（不要用代码围栏包裹全文），必须包含以下章节（标题逐字一致）：
   ## 一句话定位
   ## 适用场景
   ## 不适用 / 风险
   ## 参数说明
   ## 指标与信号逻辑
   ## 买卖规则（与策略库对齐）
   ## 周期与频率
   ## Agent 使用指引
6. 「Agent 使用指引」需说明：用户问什么时应引用本策略、probe/回测前要确认哪些参数与周期
7. 用中文，简洁可检索，避免空话与重复 JSON 原文`

func (r *Runner) phaseCognitionCompose(
	ctx context.Context,
	flow *Flow,
	_ tools.Context,
	_ func(name, status, summary string),
) error {
	body, err := r.synthesizeCognitionBody(ctx, flow)
	if err != nil {
		return fmt.Errorf("策略认知合成失败：%w", err)
	}
	flow.KBDraft = assembleAgentCognitionDoc(flow, body)
	flow.Phase = PhaseCognitionSaveKB
	flow.touch()
	return nil
}

func (r *Runner) synthesizeCognitionBody(ctx context.Context, flow *Flow) (string, error) {
	label := cognitionLabel(flow)
	user := fmt.Sprintf(`请为以下策略撰写 Agent 策略认知正文。

策略名称：%s
策略库类型：%s
signal_id：%s

策略库 JSON（权威）：
%s`,
		label,
		strings.TrimSpace(flow.CatalogType),
		catalogField(flow.CatalogRaw, "signal_id", "id"),
		catalogJSON(flow.CatalogRaw),
	)
	if r != nil && r.ComposeLLM != nil {
		resp, err := r.ComposeLLM.Chat(ctx, []llm.Message{
			{Role: llm.RoleSystem, Content: cognitionComposeSystemPrompt},
			{Role: llm.RoleUser, Content: user},
		}, nil, 0, 0)
		if err != nil {
			return "", err
		}
		if text := strings.TrimSpace(llm.VisibleAssistantContent(resp.Content, resp.ReasoningContent)); text != "" {
			return text, nil
		}
		return "", fmt.Errorf("LLM 返回空内容")
	}
	return fallbackCognitionBody(flow), nil
}

func assembleAgentCognitionDoc(flow *Flow, synthesizedBody string) string {
	label := cognitionLabel(flow)
	now := time.Now().UTC().Format(time.RFC3339)
	signalID := catalogField(flow.CatalogRaw, "signal_id", "id")
	var b strings.Builder
	fmt.Fprintf(&b, "---\n")
	fmt.Fprintf(&b, "doc_type: strategy_agent_cognition\n")
	fmt.Fprintf(&b, "strategy_name: %s\n", label)
	fmt.Fprintf(&b, "catalog_type: %s\n", flow.CatalogType)
	if signalID != "" {
		fmt.Fprintf(&b, "signal_id: %s\n", signalID)
	}
	fmt.Fprintf(&b, "generated_at: %s\n", now)
	fmt.Fprintf(&b, "source: geegoo_catalog + llm\n")
	fmt.Fprintf(&b, "agent_use: inject when user asks about this strategy, before probe/backtest\n")
	fmt.Fprintf(&b, "---\n\n")
	fmt.Fprintf(&b, "# %s · Agent 策略认知\n\n", label)
	fmt.Fprintf(&b, "> 供 Agent 注入上下文；**执行信号以策略库为准**。\n\n")
	b.WriteString(strings.TrimSpace(synthesizedBody))
	fmt.Fprintf(&b, "\n\n---\n\n## 附录 · 策略库原文\n\n")
	fmt.Fprintf(&b, "```json\n%s\n```\n", catalogJSON(flow.CatalogRaw))
	return strings.TrimSpace(b.String())
}

func fallbackCognitionBody(flow *Flow) string {
	label := cognitionLabel(flow)
	info := catalogField(flow.CatalogRaw, "info", "description", "brief")
	brief := catalogField(flow.CatalogRaw, "brief", "summary")
	var b strings.Builder
	fmt.Fprintf(&b, "## 一句话定位\n\n%s\n\n", nonEmpty(brief, label+" 策略（策略库 combination 定义）"))
	fmt.Fprintf(&b, "## 适用场景\n\n")
	fmt.Fprintf(&b, "用户需要了解或 probe/回测 **%s** 时；适用周期见策略库 frequency 字段。\n\n", label)
	fmt.Fprintf(&b, "## 不适用 / 风险\n\n")
	fmt.Fprintf(&b, "未配置 LLM 合成时仅为占位摘要；执行前请以策略库 JSON 与 probe 结果为准。\n\n")
	fmt.Fprintf(&b, "## 参数说明\n\n")
	fmt.Fprintf(&b, "详见策略库 buy_signal / sell_signal 中各 index 的 param 字段。\n\n")
	fmt.Fprintf(&b, "## 指标与信号逻辑\n\n")
	if info != "" {
		b.WriteString(info)
		b.WriteString("\n\n")
	} else {
		b.WriteString("_（策略库未提供 info 字段）_\n\n")
	}
	fmt.Fprintf(&b, "## 买卖规则（与策略库对齐）\n\n")
	fmt.Fprintf(&b, "以策略库 JSON 中 buy_signal、sell_signal 为准；type 字段区分 signal（触发）与 flag（过滤）。\n\n")
	fmt.Fprintf(&b, "## 周期与频率\n\n")
	fmt.Fprintf(&b, "%s\n\n", catalogField(flow.CatalogRaw, "frequency"))
	fmt.Fprintf(&b, "## Agent 使用指引\n\n")
	fmt.Fprintf(&b, "- 用户询问「%s 是什么 / 怎么用 / 什么参数」时检索本条目\n", label)
	fmt.Fprintf(&b, "- probe 前确认 code、frequency、months_back；回测前确认 position_mode 等账户参数\n")
	return strings.TrimSpace(b.String())
}

func cognitionLabel(flow *Flow) string {
	if flow == nil {
		return ""
	}
	if label := strings.TrimSpace(flow.CatalogLabel); label != "" {
		return label
	}
	return strings.TrimSpace(flow.StrategyQuery)
}

func catalogField(raw map[string]any, keys ...string) string {
	if raw == nil {
		return ""
	}
	for _, k := range keys {
		if s := catalogStringValue(raw[k]); s != "" {
			return s
		}
	}
	return ""
}

func nonEmpty(primary, fallback string) string {
	if s := strings.TrimSpace(primary); s != "" {
		return s
	}
	return strings.TrimSpace(fallback)
}
