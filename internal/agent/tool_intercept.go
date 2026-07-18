package agent

import (
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func filterInteractiveSchemas(schemas []llm.ToolSchema) []llm.ToolSchema {
	if len(schemas) == 0 {
		return schemas
	}
	out := make([]llm.ToolSchema, 0, len(schemas))
	for _, schema := range schemas {
		if tools.IsWorkflowExclusiveTool(schema.Name) {
			continue
		}
		out = append(out, schema)
	}
	return out
}

func (l *Loop) interceptToolCall(call llm.ToolCall, toolCtx tools.Context) (tools.Result, bool) {
	if !toolCtx.Interactive {
		return tools.Result{}, false
	}
	switch call.Name {
	case "read_working_state":
		return tools.Result{
			Status: tools.StatusSkip,
			Summary: "read_working_state 仅用于 geegoo run 盘前/盘中/盘后 workflow；" +
				"chat 请用 recall(query=关键词) 或查看本会话 Tool 活动。",
			Data: map[string]any{
				"tool": call.Name, "intercepted": true, "reason": "chat_use_recall",
			},
		}, true
	}
	if tools.IsWorkflowExclusiveTool(call.Name) {
		return tools.Result{
			Status: tools.StatusSkip,
			Summary: fmt.Sprintf("%s 仅用于 geegoo run workflow，interactive chat 中不可用。", call.Name),
			Data: map[string]any{
				"tool": call.Name, "intercepted": true, "reason": "workflow_exclusive",
			},
		}, true
	}
	return tools.Result{}, false
}
