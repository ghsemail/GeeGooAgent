package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const maxToolRounds = 8

// TurnResult is the outcome of one user turn.
type TurnResult struct {
	AssistantText string
	Failed        bool
	Error         string
}

// ReActLoop runs plan → act → observe for one chat turn.
type ReActLoop struct {
	gateway       *llm.Gateway
	executor      *Executor
	maxToolRounds int
}

// NewReActLoop creates a ReAct loop.
func NewReActLoop(gateway *llm.Gateway, executor *Executor) *ReActLoop {
	return &ReActLoop{
		gateway:       gateway,
		executor:      executor,
		maxToolRounds: maxToolRounds,
	}
}

// RunTurn executes one user message through LLM + tools.
func (l *ReActLoop) RunTurn(
	session *Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
) TurnResult {
	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: userText})
	messages := session.LLMMessages()

	for round := 0; round < l.maxToolRounds; round++ {
		session.StepCounter++
		step := session.StepCounter
		_ = step

		resp, err := l.gateway.Chat(messages, schemas, session.ID, step)
		if err != nil {
			msg := fmt.Sprintf("模型调用失败: %v", err)
			return TurnResult{AssistantText: msg, Failed: true, Error: err.Error()}
		}

		if len(resp.ToolCalls) == 0 {
			text := strings.TrimSpace(resp.Content)
			if text == "" {
				text = "（无文本回复）"
			}
			session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: text})
			return TurnResult{AssistantText: text}
		}

		assistant := llm.Message{
			Role:    llm.RoleAssistant,
			Content: resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		session.AppendMessage(assistant)
		messages = append(messages, assistant)

		for _, call := range resp.ToolCalls {
			toolCtx.Step = step
			result := l.executor.Execute(tools.CallRequest{
				Name:      call.Name,
				Arguments: call.Arguments,
			}, toolCtx)

			toolMsg := llm.Message{
				Role:       llm.RoleTool,
				Content:    toolResultContent(result),
				ToolCallID: call.ID,
			}
			session.AppendMessage(toolMsg)
			messages = append(messages, toolMsg)
		}
	}

	msg := "已达到单轮 Tool 调用上限，请缩小问题范围后重试。"
	return TurnResult{AssistantText: msg, Failed: true, Error: "max_tool_rounds"}
}

func toolResultContent(result tools.Result) string {
	payload := map[string]any{
		"status":  result.Status,
		"summary": result.Summary,
	}
	if result.Data != nil {
		payload["data"] = result.Data
	}
	raw, _ := json.Marshal(payload)
	text := string(raw)
	if len(text) > 6000 {
		return text[:6000]
	}
	return text
}
