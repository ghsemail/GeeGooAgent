package chat

import "strings"

// extractStrategyQuery strips workflow trigger words and returns the strategy name(s).
func extractStrategyQuery(text string) string {
	repl := strings.NewReplacer(
		"帮我生成", " ", "生成策略档案", " ", "的策略档案", " ", "策略档案", " ",
		"生成策略认知", " ", "策略认知", " ", "策略开发", " ",
		"学习一下", " ", "学习", " ", "了解", " ",
		"整理到知识库", " ", "写入知识库", " ", "存入知识库", " ",
		"帮我", " ", "请", " ", "workflow", " ", "Workflow", " ",
	)
	clean := strings.TrimSpace(repl.Replace(text))
	if clean == "" {
		return strings.TrimSpace(text)
	}
	if parts := splitStrategyQueries(clean); len(parts) >= 1 && len(parts) <= 2 {
		return strings.Join(parts, " ")
	}
	return clean
}
