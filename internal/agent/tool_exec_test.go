package agent_test

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestToolExecTimeout(t *testing.T) {
	registry := tools.NewRegistry()
	registry.Register(tools.Tool{
		Name: "hang",
		Handle: func(ctx tools.Context, args map[string]any) tools.Result {
			<-ctx.GoContext().Done()
			return tools.Result{Status: tools.StatusError, Summary: ctx.GoContext().Err().Error()}
		},
	})
	exec := agent.NewToolExec(runtime.NewExecutor(registry))
	exec.SetTimeout(30 * time.Millisecond)

	result := exec.Execute(context.Background(), tools.CallRequest{Name: "hang"}, tools.Context{})
	if result.Status != tools.StatusError {
		t.Fatalf("status=%s summary=%q", result.Status, result.Summary)
	}
	if result.Summary == "" {
		t.Fatal("expected timeout error summary")
	}
}
