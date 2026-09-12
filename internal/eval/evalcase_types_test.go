package eval

import "testing"

func TestClampSessionCleanupAlwaysBeforeRun(t *testing.T) {
	for _, raw := range []string{"", "never", "after_run", "before_run", "BEFORE_RUN"} {
		if got := ClampSessionCleanup(raw); got != DefaultEvalSessionCleanup {
			t.Fatalf("ClampSessionCleanup(%q)=%q want %q", raw, got, DefaultEvalSessionCleanup)
		}
	}
}

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

func TestSyncLegacyUtterancesSkipsOnClarifyTurns(t *testing.T) {
	opts := TurnPlanCaseOptions{
		Category: "turn_plan",
		Dialogue: []EvalDialogueTurn{
			{Role: "user", Text: "帮我分析一下腾讯的价格走势"},
			{Role: "user", Text: "帮我看看哪些策略适合腾讯"},
			{Role: "user", Text: "组合信号（多指标共振，推荐稳健）", OnClarify: true},
			{Role: "user", Text: "帮我用这些策略回测一下", Judge: true},
		},
	}.SyncLegacyUtterances()

	if len(opts.SetupMessages) != 2 {
		t.Fatalf("setup=%v", opts.SetupMessages)
	}
	if opts.Message != "帮我用这些策略回测一下" {
		t.Fatalf("message=%q", opts.Message)
	}
}
