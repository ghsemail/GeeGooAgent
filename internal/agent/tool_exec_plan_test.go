package agent

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestExecuteBatchEmitsPlanProposed(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(tools.Tool{
		Name: "create_bot",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			return tools.Result{Status: tools.StatusOK, Summary: "ok"}
		},
	})
	exec := runtime.NewExecutor(reg)
	te := NewToolExec(exec)
	te.SetPlanGate(true)

	var events []string
	calls := []llm.ToolCall{{Name: "create_bot", Arguments: map[string]any{"name": "x"}}}
	te.ExecuteBatch(context.Background(), calls, tools.Context{Interactive: true, Approved: true}, 1, func(event string, data map[string]any) {
		events = append(events, event)
	})
	if len(events) == 0 {
		t.Fatalf("events=%v", events)
	}
}
