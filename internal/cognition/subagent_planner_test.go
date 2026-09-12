package cognition_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
)

func TestSubAgentPlannerSkipsClassify(t *testing.T) {
	t.Parallel()
	plan := cognition.SubAgentPlanner{}.Plan(cognition.PlanInput{
		UserText: "分析阿里巴巴最近股价（独立子任务）",
	})
	if plan.ClassifyFailed() {
		t.Fatalf("classify failed: %+v", plan)
	}
	if plan.Domain != cognition.DomainStockAnalysis {
		t.Fatalf("domain=%s", plan.Domain)
	}
	if plan.Mode != cognition.ModeGather {
		t.Fatalf("mode=%s", plan.Mode)
	}
	if len(plan.Skills) == 0 {
		t.Fatal("expected stock skills")
	}
}
