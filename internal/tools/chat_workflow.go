package tools

import "strings"

// ChatWorkflowTurn executes one chat workflow template step (signal_diagnose, strategy_dev, etc.).
type ChatWorkflowTurn func(ctx Context, userMessage string) Result

func registerChatWorkflowTool(r *Registry, deps Deps) {
	if r == nil {
		return
	}
	r.Register(Tool{
		Name: "run_chat_workflow_turn",
		Description: "[Deprecated — prefer skill_view + ordinary tools.] 推进或启动 Chat 工作流技能（信号诊断、策略开发等）。" +
			"仅当 chat_open_flow_tool 启用时注册。将 user_message 设为用户的原话（或「继续」「重试失败」）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"user_message": map[string]any{
					"type":        "string",
					"description": "本回合用户消息或续跑指令（继续 / 重试失败）。",
				},
			},
			"required": []any{"user_message"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			if deps.ChatWorkflowTurn == nil {
				return Result{
					Status:  StatusError,
					Summary: "run_chat_workflow_turn 未配置",
				}
			}
			msg := strings.TrimSpace(strArg(args, "user_message", ""))
			if msg == "" {
				return Result{Status: StatusError, Summary: "user_message required"}
			}
			return deps.ChatWorkflowTurn(ctx, msg)
		},
	})
}
