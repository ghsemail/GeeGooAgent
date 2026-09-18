package chatprompt

import "strings"

// PlanOverviewMarkdown is the intro shown on Memory → Plan.
func PlanOverviewMarkdown() string {
	return strings.TrimSpace(`Turn Plan 在每轮用户消息后运行：**Classify → Plan fragment → ReAct**。

## 软约束引导（Turn Plan 机制）

每轮 classify 之后，向 system 注入一段 **Plan fragment**，包含 domain / mode / act、**有序 plan steps**、**preferred tools**（推荐工具，非白名单）。

| 行为 | 说明 |
|------|------|
| **不裁剪 tool schema** | ReAct 仍能看到完整工具列表；plan 只是「建议路线」 |
| **优先按 steps 执行** | 模型应按步骤顺序调用推荐工具完成用户意图 |
| **允许偏离** | 仅当会话上下文或用户原话明确要求时，才可跳过某步或换工具 |
| **缺槽位先 clarify** | 标的/策略/模式不明时，先 clarify 再跑重工具 |

与「硬路由 / 固定 SOP 脚本」不同：没有 deterministic shortcut，最终仍由 ReAct 选 tool call；plan 提供 **可读的执行清单**，并同步到 SSE turn_plan.plan_steps 字段。

## 链路

- **Classify**：LLM 输出 domain / mode / act（见 Classify Prompt）
- **Plan fragment**：steps + preferred tools + execution profile（见 Plan Fragment / Plan Steps）
- **Playbook**：domain 映射后在 **Skills** 页查看 SKILL.md 正文
- **运行时**：Chat / Loop 的 turn_plan SSE 可见本轮 plan_steps

## 回测 vs 生成

- **回测** → 唯一 playbook strategy-backtest-run → run_strategy_backtest（回测 ≠ 生成）
- **生成策略** → 用户明确「生成/设计/出方案」→ generate_*（见 Tool 路由）`)
}
