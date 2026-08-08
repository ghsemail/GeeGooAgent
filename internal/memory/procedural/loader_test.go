package procedural

import (
	"strings"
	"testing"
)

func TestMatchPreMarketSkill(t *testing.T) {
	t.Parallel()
	l := NewLoader("../../../skills")
	matched := l.Match("帮我生成 pre_market_stock 盘前报告", 2)
	if len(matched) == 0 {
		t.Fatal("expected pre_market_stock skill match")
	}
	found := false
	for _, sk := range matched {
		if sk.Name == "pre_market_stock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("pre_market_stock not in %v", matched)
	}
}

func TestFormatSkills(t *testing.T) {
	t.Parallel()
	text := Format([]Skill{{Name: "demo", Body: "step 1"}})
	if text == "" || !strings.Contains(text, "demo") {
		t.Fatalf("unexpected format: %q", text)
	}
}
