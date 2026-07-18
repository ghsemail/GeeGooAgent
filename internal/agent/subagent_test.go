package agent_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestDelegateTaskIsolatesSession(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "子任务完成：腾讯 00700.HK。"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "search_code", Description: "search",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	sub := agent.NewSubAgent(agent.SubAgentConfig{
		Gateway: gateway, Executor: runtime.NewExecutor(registry), Registry: registry,
		MaxSteps: 5, ChatToolNames: func() []string { return []string{"search_code"} },
	})
	agent.RegisterDelegateTask(registry, sub)

	parentSession := runtime.NewSession()
	parentBefore := len(parentSession.Messages)

	result := registry.Execute(tools.CallRequest{
		Name: "delegate_task",
		Arguments: map[string]any{
			"task": "查腾讯代码",
		},
	}, tools.Context{SessionID: parentSession.ID, DelegateDepth: 0})

	if result.Status != tools.StatusOK {
		t.Fatalf("status=%s summary=%q", result.Status, result.Summary)
	}
	if answer, _ := result.Data["answer"].(string); !strings.Contains(answer, "00700") {
		t.Fatalf("answer=%q", answer)
	}
	if len(parentSession.Messages) != parentBefore {
		t.Fatalf("parent session mutated: before=%d after=%d", parentBefore, len(parentSession.Messages))
	}
}

func TestDelegateTaskRejectsNesting(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	sub := agent.NewSubAgent(agent.SubAgentConfig{
		Gateway: llm.NewGateway(&llm.MockProvider{}, llm.GatewayConfig{}),
		Executor: runtime.NewExecutor(registry), Registry: registry, MaxSteps: 5,
	})
	agent.RegisterDelegateTask(registry, sub)

	result := registry.Execute(tools.CallRequest{
		Name: "delegate_task", Arguments: map[string]any{"task": "x"},
	}, tools.Context{SessionID: "s1", DelegateDepth: 1})

	if result.Status != tools.StatusError || !strings.Contains(result.Summary, "nested") {
		t.Fatalf("result=%+v", result)
	}
}

func TestDelegateTaskEmitsProgress(t *testing.T) {
	t.Parallel()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "done"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	registry := tools.NewRegistry()
	sub := agent.NewSubAgent(agent.SubAgentConfig{
		Gateway: gateway, Executor: runtime.NewExecutor(registry), Registry: registry, MaxSteps: 5,
	})
	agent.RegisterDelegateTask(registry, sub)

	var events []string
	registry.Execute(tools.CallRequest{
		Name: "delegate_task", Arguments: map[string]any{"task": "hello"},
	}, tools.Context{
		SessionID: "s1",
		Progress: func(event string, _ map[string]any) {
			events = append(events, event)
		},
		Ctx: context.Background(),
	})

	if len(events) < 2 || events[0] != "subagent_start" || events[len(events)-1] != "subagent_end" {
		t.Fatalf("events=%v", events)
	}
}
