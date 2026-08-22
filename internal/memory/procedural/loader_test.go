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

func TestMatchKnowledgeBaseSkill(t *testing.T) {
	t.Parallel()
	l := NewLoader("../../../skills")
	matched := l.Match("按知识库里的 4 小时 MACD 策略说明一下", 5)
	found := false
	for _, sk := range matched {
		if sk.Name == "knowledge-base" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected knowledge-base in %v", matched)
	}
	for _, sk := range l.Match("腾讯现在股价多少", 5) {
		if sk.Name == "knowledge-base" {
			t.Fatal("quote question should not inject knowledge-base")
		}
	}
}

func TestFormatSkills(t *testing.T) {
	t.Parallel()
	text := Format([]Skill{{Name: "demo", Body: "step 1"}})
	if text == "" || !strings.Contains(text, "demo") {
		t.Fatalf("unexpected format: %q", text)
	}
}
