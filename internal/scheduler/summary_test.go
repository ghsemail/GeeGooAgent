package scheduler_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/scheduler"
)

func TestShouldNotifyJob(t *testing.T) {
	t.Parallel()
	if !scheduler.ShouldNotifyJobForTest(scheduler.Job{Skill: "premarket_stock", Platform: "feishu"}) {
		t.Fatal("expected notify")
	}
	if scheduler.ShouldNotifyJobForTest(scheduler.Job{Skill: "premarket_market", Platform: "feishu"}) {
		t.Fatal("market job should not notify")
	}
}
