package tools

import "testing"

func TestIsWorkflowExclusiveTool(t *testing.T) {
	t.Parallel()
	if !IsWorkflowExclusiveTool("read_working_state") {
		t.Fatal("read_working_state should be workflow-exclusive")
	}
	if !IsWorkflowExclusiveTool("create_pre_market_report") {
		t.Fatal("create_pre_market_report should be workflow-exclusive")
	}
	if IsWorkflowExclusiveTool("get_bot_yesterday_attitude") {
		t.Fatal("get_bot_yesterday_attitude is shared with market toolset")
	}
	if IsWorkflowExclusiveTool("recall") {
		t.Fatal("recall is a chat memory tool")
	}
}
