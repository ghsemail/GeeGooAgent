package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Runner) phaseCognitionCompose(
	ctx context.Context,
	flow *Flow,
	_ tools.Context,
	_ func(name, status, summary string),
) error {
	body, err := r.synthesizeCognitionBody(ctx, flow)
	if err != nil {
		return fmt.Errorf("策略档案合成失败：%w", err)
	}
	flow.KBDraft = assembleAgentCognitionDoc(flow, body)
	flow.Phase = PhaseCognitionSaveKB
	flow.touch()
	return nil
}

func (r *Runner) synthesizeCognitionBody(ctx context.Context, flow *Flow) (string, error) {
	label := cognitionLabel(flow)
	user := fmt.Sprintf(`请为以下策略撰写策略档案正文。

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
			{Role: llm.RoleSystem, Content: cognitionComposeSystemPrompt()},
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
		v := raw[k]
		switch t := v.(type) {
		case []any:
			parts := make([]string, 0, len(t))
			for _, item := range t {
				if s := catalogStringValue(item); s != "" {
					parts = append(parts, s)
				}
			}
			if len(parts) > 0 {
				return strings.Join(parts, ", ")
			}
		default:
			if s := catalogStringValue(v); s != "" {
				return s
			}
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
