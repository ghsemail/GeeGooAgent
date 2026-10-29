package runtimeapi

import (
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestEvalChatTimeoutClarifyPadding(t *testing.T) {
	opts := eval.TurnPlanCaseOptions{
		ExpectMode:   "clarify",
		ClarifyReply: "先问答",
	}.Normalize()
	got := evalChatTimeout(opts)
	want := 10*time.Minute + time.Duration(eval.EvalClarifyWaitBudgetSec)*time.Second
	if got != want {
		t.Fatalf("timeout=%v want %v", got, want)
	}
}

func TestEvalChatTimeoutRespectsExplicitWait(t *testing.T) {
	opts := eval.TurnPlanCaseOptions{
		WaitTimeoutSec: 900,
		ExpectMode:     "clarify",
		ClarifyReply:   "先问答",
	}.Normalize()
	got := evalChatTimeout(opts)
	want := 900*time.Second + time.Duration(eval.EvalClarifyWaitBudgetSec)*time.Second
	if got != want {
		t.Fatalf("timeout=%v want %v", got, want)
	}
}
