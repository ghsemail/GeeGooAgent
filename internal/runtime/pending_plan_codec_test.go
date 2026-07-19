package runtime_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestPendingPlanMapRoundTrip(t *testing.T) {
	t.Parallel()
	orig := &runtime.PendingPlan{
		Step: 3,
		ToolCalls: []llm.ToolCall{{
			ID: "c1", Name: "create_dca_bot", Arguments: map[string]any{"botname": "t"},
		}},
	}
	m := runtime.PendingPlanToMap(orig)
	got := runtime.PendingPlanFromMap(m)
	if got == nil || got.Step != 3 || len(got.ToolCalls) != 1 {
		t.Fatalf("round trip failed: %+v", got)
	}
	if got.ToolCalls[0].Name != "create_dca_bot" {
		t.Fatalf("tool=%+v", got.ToolCalls[0])
	}
}

func TestPendingPlanFromMapEmpty(t *testing.T) {
	t.Parallel()
	if runtime.PendingPlanFromMap(nil) != nil {
		t.Fatal("expected nil")
	}
	if runtime.PendingPlanFromMap(map[string]any{"tool_calls": []any{}}) != nil {
		t.Fatal("expected nil for empty calls")
	}
}
