package scheduler_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestVerdictForTestPass(t *testing.T) {
	t.Parallel()
	r := &workflow.SupervisorReport{Verdict: workflow.VerdictPass}
	if scheduler.VerdictForTest(r) != "pass" {
		t.Fatal("expected pass")
	}
	if scheduler.VerdictForTest(nil) != "unknown" {
		t.Fatal("expected unknown for nil")
	}
}
