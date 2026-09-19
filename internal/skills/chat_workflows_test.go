package skills

import "testing"

func TestChatWorkflowCatalog(t *testing.T) {
	items := ChatWorkflowCatalog()
	if len(items) == 0 {
		t.Fatal("empty catalog")
	}
	if items[0]["id"] != SkillGenerateStrategyArchive {
		t.Fatalf("id=%v", items[0]["id"])
	}
}

func TestCanonicalGenerateStrategyArchive(t *testing.T) {
	if CanonicalName("generate_strategy_cognition") != SkillGenerateStrategyArchive {
		t.Fatal("legacy id should map to generate_strategy_archive")
	}
	spec, ok := Default().Get("generate_strategy_cognition")
	if !ok || spec.Name != SkillGenerateStrategyArchive {
		t.Fatalf("lookup alias: ok=%v name=%s", ok, spec.Name)
	}
	if DisplayName(spec) != "生成策略档案" {
		t.Fatalf("display=%q", DisplayName(spec))
	}
}

func TestIsChatWorkflowSkillStrategyWorkflows(t *testing.T) {
	if !IsChatWorkflowSkill("generate_strategy_cognition") {
		t.Fatal("expected legacy cognition alias")
	}
	if !IsChatWorkflowSkill(SkillGenerateStrategyArchive) {
		t.Fatal("expected generate_strategy_archive")
	}
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
