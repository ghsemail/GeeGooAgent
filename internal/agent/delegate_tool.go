package agent

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// RegisterDelegateTask adds the delegate_task tool for spawning isolated sub-agents.
func RegisterDelegateTask(r *tools.Registry, delegate tools.TaskDelegator) {
	if r == nil || delegate == nil {
		return
	}
	r.Register(tools.Tool{
		Name: "delegate_task",
		Description: "将复杂子任务委托给独立子 Agent（独立 ReAct 回合，不写入主会话历史）。" +
			"适合：多步调研、需多轮工具的单路子任务。单标的简单查价/分析请直接调业务工具。" +
			"子 Agent 不能再嵌套 delegate。",
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
			return delegate.DelegateTask(ctx, task, background, maxSteps)
		},
	})
}

// RegisterDelegateTasks adds batch parallel delegation (bounded by delegate_max_parallel).
func RegisterDelegateTasks(r *tools.Registry, delegate tools.TaskDelegator) {
	if r == nil || delegate == nil {
		return
	}
	r.Register(tools.Tool{
		Name: "delegate_tasks",
		Description: "并行委托 1–8 个独立子 Agent（等同 Cursor Task：同轮多次并行子任务，不写入主会话历史）。" +
			"首选场景：用户在同一句话里提到 2+ 个互不依赖的公司/标的（如「分析腾讯和阿里巴巴股价」）——一次调用 tasks[]，每路一项，平台并行执行。" +
			"每路 task 须自洽（含公司名/标的与要完成的分析）。返回后主 Agent 必须在最终回复中逐标的写出具体现价/分析要点并做简要对比，禁止只写「已委派」。" +
			"单标的简单查价/走势请直接 search_code / get_current_price / get_mcp_analysis，勿 delegate。子 Agent 不能再嵌套 delegate。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"tasks": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"task": map[string]any{
								"type":        "string",
								"description": "子任务描述（必填）",
							},
							"context": map[string]any{
								"type":        "string",
								"description": "可选背景信息",
							},
							"max_steps": map[string]any{
								"type":        "integer",
								"description": "该子任务最大轮数（可选）",
							},
						},
						"required": []any{"task"},
					},
					"description": "子任务列表（1–8 项）",
				},
			},
			"required": []any{"tasks"},
		},
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			specs, err := parseBatchDelegateTasks(args["tasks"])
			if err != nil {
				return tools.Result{Status: tools.StatusError, Summary: err.Error(), ExitCode: 1}
			}
			return delegate.DelegateTasks(ctx, specs)
		},
	})
}

func parseBatchDelegateTasks(raw any) ([]tools.BatchDelegateTask, error) {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		return nil, fmt.Errorf("delegate_tasks: tasks required")
	}
	specs := make([]tools.BatchDelegateTask, 0, len(items))
	for i, item := range items {
		obj, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("delegate_tasks: tasks[%d] must be object", i)
		}
		task := strArg(obj, "task", "")
		if task == "" {
			return nil, fmt.Errorf("delegate_tasks: tasks[%d].task required", i)
		}
		specs = append(specs, tools.BatchDelegateTask{
			Task:       task,
			Background: strArg(obj, "context", ""),
			MaxSteps:   intArg(obj, "max_steps", 0),
		})
	}
	return specs, nil
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
