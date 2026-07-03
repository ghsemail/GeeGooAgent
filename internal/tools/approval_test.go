package tools_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestApprovalRequiredForMutatingTools(t *testing.T) {
	t.Parallel()
	for _, name := range []string{"create_dca_bot", "update_grid_bot", "delete_smart_trade", "switch_bot", "create_pre_market_report"} {
		if !tools.ApprovalRequired(name) {
			t.Fatalf("%s should require approval", name)
		}
	}
	for _, name := range []string{"list_smart_trades", "get_current_price", "search_code", "get_mcp_analysis"} {
		if tools.ApprovalRequired(name) {
			t.Fatalf("%s should not require approval", name)
		}
	}
}

func TestApprovalGateBlocksInteractiveUnapproved(t *testing.T) {
	t.Parallel()
	called := false
	gated := tools.ApprovalGate("create_dca_bot", func(ctx tools.Context, args map[string]any) tools.Result {
		called = true
		return tools.Result{Status: tools.StatusOK, Summary: "created"}
	})
	res := gated(tools.Context{Interactive: true, Approved: false}, nil)
	if called {
		t.Fatal("handler should not run when unapproved in interactive")
	}
	if res.Status != tools.StatusSkip {
		t.Fatalf("expected skip, got %s", res.Status)
	}
	if !res.Data["approval_required"].(bool) {
		t.Fatal("expected approval_required flag")
	}
}

func TestApprovalGateAllowsApprovedInteractive(t *testing.T) {
	t.Parallel()
	called := false
	gated := tools.ApprovalGate("delete_smart_trade", func(ctx tools.Context, args map[string]any) tools.Result {
		called = true
		return tools.Result{Status: tools.StatusOK, Summary: "deleted"}
	})
	res := gated(tools.Context{Interactive: true, Approved: true}, nil)
	if !called {
		t.Fatal("handler should run when approved")
	}
	if res.Status != tools.StatusOK {
		t.Fatalf("expected ok, got %s", res.Status)
	}
}

func TestApprovalGateAllowsWorkflowNonInteractive(t *testing.T) {
	t.Parallel()
	called := false
	gated := tools.ApprovalGate("create_pre_market_report", func(ctx tools.Context, args map[string]any) tools.Result {
		called = true
		return tools.Result{Status: tools.StatusOK, Summary: "reported"}
	})
	res := gated(tools.Context{Interactive: false}, nil) // workflow path
	if !called {
		t.Fatal("handler should run in non-interactive workflow")
	}
	if res.Status != tools.StatusOK {
		t.Fatalf("expected ok, got %s", res.Status)
	}
}
