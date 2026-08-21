package context

import "strings"

// BudgetReminderFragment warns the model when context budget is nearly exhausted.
func BudgetReminderFragment(usedRatio float64) Fragment {
	if usedRatio < 0.8 {
		return StaticFragment{}
	}
	msg := "[Budget] 上下文已使用较多，请简洁答复并避免重复调用工具。"
	if usedRatio >= 1 {
		msg = "[Budget] 上下文预算已满，请给出简短总结并结束本轮。"
	}
	return StaticFragment{
		K:    KindBudgetReminder,
		Text: msg,
		Prio: 5,
	}
}

// SystemRulesFragment wraps merged AGENTS.md / profile text for fragment pipeline.
func SystemRulesFragment(text string) Fragment {
	text = strings.TrimSpace(text)
	if text == "" {
		return StaticFragment{}
	}
	return StaticFragment{
		K:    KindSystemRules,
		Text: text,
		Prio: 10,
	}
}
