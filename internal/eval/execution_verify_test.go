package eval_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestVerifyExecutionProfilesMatchLiveFailures(t *testing.T) {
	technicalChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_current_price"}},
				{Turn: 2, Tools: []string{"get_single_prompt_template", "get_mcp_analysis"}},
			},
		},
	}
	res := eval.VerifyExecution(technicalChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockTechnicalFull,
	})
	if !res.Passed {
		t.Fatalf("technical chain should pass profile check: %s", res.Detail)
	}

	colloquialChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_mcp_analysis"}},
				{Turn: 2, Tools: []string{"get_single_prompt_template", "get_current_price", "get_mcp_analysis"}},
			},
		},
	}
	res = eval.VerifyExecution(colloquialChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockContextFollowup,
	})
	if !res.Passed {
		t.Fatalf("colloquial ref should pass profile check: %s", res.Detail)
	}

	priceChat := &chatsession.ChatSession{
		Metadata: map[string]any{
			"turn_tools_trace": []chatsession.TurnToolsEntry{
				{Turn: 1, Tools: []string{"search_code", "get_current_price"}},
			},
		},
	}
	res = eval.VerifyExecution(priceChat, eval.ExpectExecutionSpec{
		Profile: domaincatalog.ProfileStockPriceSnapshot,
	})
	if !res.Passed {
		t.Fatalf("price snapshot should pass profile check: %s", res.Detail)
	}
}
