package eval

import "testing"

func TestSyncLegacyUtterancesFromDialogue(t *testing.T) {
	opts := TurnPlanCaseOptions{
		Category: "turn_plan",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我查一下腾讯的股价"},
			{Role: "user", Text: "可以，分析下技术面的价格和K线图", Judge: true},
		},
	}.SyncLegacyUtterances()

	if len(opts.SetupMessages) != 1 || opts.SetupMessages[0] != "帮我查一下腾讯的股价" {
		t.Fatalf("setup=%v", opts.SetupMessages)
	}
	if opts.Message != "可以，分析下技术面的价格和K线图" {
		t.Fatalf("message=%q", opts.Message)
	}
}
