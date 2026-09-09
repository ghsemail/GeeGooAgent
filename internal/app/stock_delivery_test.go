package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestUserStockDeliveryOKRequiresFeishuWhenNotifyEnabled(t *testing.T) {
	t.Parallel()
	result := workflow.RunResult{
		Status:     "completed",
		Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass},
		Working:    &memory.PreMarketWorking{Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Status: "reported", Code: "00700.HK"},
		}},
	}
	ok, reason := userStockDeliveryOK("premarket_stock", "HK", result, nil, true, false, "feishu_send_failed")
	if ok || reason != "feishu_send_failed" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
	ok, reason = userStockDeliveryOK("premarket_stock", "HK", result, nil, true, true, "")
	if !ok || reason != "" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestUserStockDeliveryOKSkipsPushOnNonTradingDay(t *testing.T) {
	t.Parallel()
	falseDay := false
	result := workflow.RunResult{
		Status:     "completed",
		Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass},
		Working:    &memory.PreMarketWorking{IsTradingDay: &falseDay},
	}
	ok, reason := userStockDeliveryOK("premarket_stock", "HK", result, nil, true, false, "notify_disabled")
	if !ok || reason != "" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestUserStockDeliveryOKAcceptsRecoverableWhenReported(t *testing.T) {
	t.Parallel()
	result := workflow.RunResult{
		Status:     "completed",
		Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictRecoverable},
		Working:    &memory.PreMarketWorking{Stocks: map[string]memory.StockWorkspace{
			"SPCX.US": {Status: "reported", Code: "SPCX.US"},
		}},
	}
	ok, reason := userStockDeliveryOK("postmarket_stock", "US", result, nil, false, false, "")
	if !ok || reason != "" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}
	t.Parallel()
	result := workflow.RunResult{
		Status:     "completed",
		Supervisor: &workflow.SupervisorReport{Verdict: workflow.VerdictPass},
		Working:    &memory.PreMarketWorking{Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {Status: "skipped", Code: "00700.HK"},
		}},
	}
	ok, reason := userStockDeliveryOK("premarket_stock", "HK", result, nil, true, false, "no_new_reports")
	if !ok || reason != "" {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}

func TestSkillDeliveryReasonForSchedulerUsesRunError(t *testing.T) {
	t.Parallel()
	ok, reason := SkillDeliveryReasonForScheduler(workflow.RunResult{}, errors.New("report delivery failed for 1 user(s)"))
	if ok || !strings.Contains(reason, "report delivery failed") {
		t.Fatalf("ok=%v reason=%q", ok, reason)
	}
}
