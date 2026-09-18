package chatprompt

import "strings"

// PlanOverviewMarkdown is the intro shown on Memory → Plan.
func PlanOverviewMarkdown() string {
	return strings.TrimSpace(`Turn Plan 在每轮用户消息后运行：**Classify → Plan fragment → ReAct**。

- **Classify**：LLM 输出 domain / mode / act（见下方 Classify Prompt）
- **Plan fragment**：Cursor 风格步骤 + preferred tools（软约束）
- **Playbook**：domain 映射后在 **Skills** 页查看 SKILL.md 正文
- **运行时**：Chat / Loop 的 turn_plan SSE 事件可见本轮 plan_steps

**回测 vs 生成**
- **回测** → 唯一 playbook strategy-backtest-run → run_strategy_backtest（回测 ≠ 生成）
- **生成策略** → 用户明确「生成/设计/出方案」→ generate_*（见 Tool 路由）`)
}
