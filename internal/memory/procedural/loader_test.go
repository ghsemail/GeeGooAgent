package procedural

import (
	"strings"
	"testing"
)

func TestMatchPreMarketSkill(t *testing.T) {
	t.Parallel()
	l := NewLoader("../../../skills")
	matched := l.Match("geegoo run premarket_stock --market CN", 10)
	if len(matched) == 0 {
		t.Fatal("expected premarket_stock skill match")
	}
	found := false
	for _, sk := range matched {
		if sk.Name == "premarket_stock" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("premarket_stock not in %v", matched)
	}
}

func TestFormatSkills(t *testing.T) {
	t.Parallel()
	text := Format([]Skill{{Name: "demo", Body: "step 1"}})
	if text == "" || !strings.Contains(text, "demo") {
		t.Fatalf("unexpected format: %q", text)
	}
}
