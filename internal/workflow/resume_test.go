package workflow_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestStepKeyAndCompletionHelpers(t *testing.T) {
	t.Parallel()
	w := memory.NewPreMarketWorking("s1", "pre_market")
	if workflow.IsStepCompleteForTest(w, "00700.HK/weekly_analysis") {
		t.Fatal("should not be complete initially")
	}
	workflow.MarkStepCompleteForTest(w, "00700.HK/weekly_analysis")
	if !workflow.IsStepCompleteForTest(w, "00700.HK/weekly_analysis") {
		t.Fatal("should be complete after mark")
	}
	// Idempotent: marking twice does not duplicate.
	before := len(w.CompletedStepKeys)
	workflow.MarkStepCompleteForTest(w, "00700.HK/weekly_analysis")
	if len(w.CompletedStepKeys) != before {
		t.Fatalf("mark not idempotent: %d -> %d", before, len(w.CompletedStepKeys))
	}
}

func TestStepKeyFallsBackToTool(t *testing.T) {
	t.Parallel()
	if workflow.StepKeyForTest("", "get_mcp_analysis") != "get_mcp_analysis" {
		t.Fatal("empty name should fall back to tool")
	}
	if workflow.StepKeyForTest("code/step", "get_mcp_analysis") != "code/step" {
		t.Fatal("named step should use name")
	}
}
