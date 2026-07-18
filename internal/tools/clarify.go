package tools

import (
	"fmt"
	"strings"
)

const (
	// ClarifyMaxChoices is the Hermes clarify tool limit.
	ClarifyMaxChoices = 4
	// ClarifyOtherLabel is appended by the UI when choices are provided.
	ClarifyOtherLabel = "其他（自行输入）"
)

// ClarifyFunc blocks until the user answers a clarify prompt (interactive chat only).
type ClarifyFunc func(question string, choices []string) (answer string, ok bool)

func registerClarifyTool(r *Registry) {
	r.Register(Tool{
		Name: "clarify",
		Description: "向用户提问以澄清意图、获取反馈或在继续前做选择。支持两种模式：" +
			"1) 多选：提供最多 4 个 choices，UI 自动追加「其他（自行输入）」；" +
			"2) 开放式：省略 choices，用户自由输入。" +
			"任务含糊、需在多个方案中选一、或决策有明确取舍时使用；不要用简单 y/n 代替写操作确认（由 approval 处理）。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"question": map[string]any{
					"type":        "string",
					"description": "呈现给用户的问题。",
				},
				"choices": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "string",
					},
					"description": "最多 4 个选项；省略则为开放式问题。",
				},
			},
			"required": []any{"question"},
		},
		Handle: handleClarify,
	})
}

func handleClarify(ctx Context, args map[string]any) Result {
	question := strings.TrimSpace(strArg(args, "question", ""))
	if question == "" {
		return Result{Status: StatusError, Summary: "question 不能为空"}
	}
	choices := normalizeClarifyChoices(args["choices"])
	if ctx.DryRun {
		return Result{
			Status:  StatusDryRun,
			Summary: "dry-run: skipped clarify",
			Data:    map[string]any{"user_response": "(dry-run)"},
		}
	}
	if !ctx.Interactive {
		return Result{Status: StatusSkip, Summary: "clarify 仅用于交互式 chat"}
	}
	if ctx.ClarifyFn == nil {
		return Result{Status: StatusError, Summary: "clarify 未配置（当前环境无用户输入回调）"}
	}
	answer, ok := ctx.ClarifyFn(question, choices)
	answer = strings.TrimSpace(answer)
	if !ok {
		return Result{
			Status:  StatusSkip,
			Summary: "用户未回答",
			Data:    map[string]any{"user_response": ""},
		}
	}
	if answer == "" {
		return Result{
			Status:  StatusSkip,
			Summary: "用户跳过",
			Data:    map[string]any{"user_response": ""},
		}
	}
	return Result{
		Status:  StatusOK,
		Summary: answer,
		Data:    map[string]any{"user_response": answer},
	}
}

func normalizeClarifyChoices(raw any) []string {
	items, ok := raw.([]any)
	if !ok || len(items) == 0 {
		if typed, ok := raw.([]string); ok {
			items = make([]any, len(typed))
			for i, s := range typed {
				items[i] = s
			}
		}
	}
	if len(items) == 0 {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		s, _ := item.(string)
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		out = append(out, s)
		if len(out) >= ClarifyMaxChoices {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ClarifyDisplayOptions returns choices plus the Hermes "Other" option when needed.
func ClarifyDisplayOptions(choices []string) []string {
	if len(choices) == 0 {
		return nil
	}
	out := append([]string(nil), choices...)
	out = append(out, ClarifyOtherLabel)
	return out
}

// ClarifyChoiceLabel returns A/B/C style label for index i.
func ClarifyChoiceLabel(i int) string {
	if i < 0 || i > 25 {
		return fmt.Sprintf("%d", i+1)
	}
	return string(rune('A' + i))
}
