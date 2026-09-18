package skills

import "testing"

func TestChatWorkflowCatalog(t *testing.T) {
	items := ChatWorkflowCatalog()
	if len(items) == 0 {
		t.Fatal("empty catalog")
	}
	if items[0]["id"] != SkillStrategyDev {
		t.Fatalf("id=%v", items[0]["id"])
	}
}

func TestIsChatWorkflowSkillStrategyDev(t *testing.T) {
	if !IsChatWorkflowSkill(SkillStrategyDev) {
		t.Fatal("expected strategy_dev")
	}
}

func TestIsChatWorkflowSkill(t *testing.T) {
	if !IsChatWorkflowSkill(SkillMultiStrategyCompare) {
		t.Fatal("expected multi_strategy_compare")
	}
	if IsChatWorkflowSkill("premarket_market") {
		t.Fatal("premarket_market is batch-only")
	}
}
