package workflow_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBotLogStepIsOptional(t *testing.T) {
	step := workflow.Step{Name: "bot_log", Tool: "get_bot_log_by_type"}
	if !workflow.OptionalStepForTest(step) {
		t.Fatal("bot log should be optional when MCP returns no permission")
	}
}
