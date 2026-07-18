package agent

import (
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// RegisterDelegateTask adds the delegate_task tool for spawning isolated sub-agents.
func RegisterDelegateTask(r *tools.Registry, sub *SubAgent) {
	if r == nil || sub == nil {
		return
	}
	r.Register(tools.Tool{
		Name: "delegate_task",
		Description: "将复杂子任务委托给独立子 Agent 执行（独立回合预算与上下文，不写入主会话历史）。" +
			"适合多步调研、并行信息收集等；子 Agent 不能再嵌套 delegate。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"task": map[string]any{
					"type":        "string",
					"description": "子 Agent 需要完成的具体任务（必填）",
				},
				"context": map[string]any{
					"type":        "string",
					"description": "可选背景信息（主会话摘要、约束等）",
				},
				"max_steps": map[string]any{
					"type":        "integer",
					"description": "子 Agent 最大 LLM↔tool 轮数（默认配置 sub_agent_max_steps）",
				},
			},
			"required": []any{"task"},
		},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			task := strArg(args, "task", "")
			background := strArg(args, "context", "")
			maxSteps := intArg(args, "max_steps", 0)
			return sub.Run(ctx, task, background, maxSteps)
		},
	})
}

func strArg(args map[string]any, key, def string) string {
	if v, ok := args[key].(string); ok && v != "" {
		return v
	}
	return def
}

func intArg(args map[string]any, key string, def int) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return def
	}
}
