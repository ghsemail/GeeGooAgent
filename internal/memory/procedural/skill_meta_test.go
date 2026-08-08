package procedural

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestClassifyPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want Kind
	}{
		{`skills/playbooks/stock-analysis/SKILL.md`, KindPlaybook},
		{`skills/premarket_market/SKILL.md`, KindWorkflow},
		{`skills/premarket_stock/SKILL.md`, KindWorkflow},
		{`skills/extensions/finance-news/SKILL.md`, KindExtension},
		{`skills/bundled/finance-news/SKILL.md`, KindExtension},
		{`/home/ubuntu/.geegoo/skills/my-skill/SKILL.md`, KindUser},
	}
	for _, tc := range cases {
		if got := ClassifyPath(tc.path); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.path, got, tc.want)
		}
	}
}

func TestClassifyProvenance(t *testing.T) {
	t.Parallel()
	if ClassifyProvenance(`skills/extensions/duckduckgo-search/SKILL.md`) != ProvenanceOptional {
		t.Fatal("extension should be optional")
	}
	if ClassifyProvenance(`skills/playbooks/x/SKILL.md`) != ProvenanceCore {
		t.Fatal("playbook should be core")
	}
}

func TestMatchSkipsExtension(t *testing.T) {
	t.Parallel()
	l := NewLoader("../../../skills")
	matched := l.Match("财经新闻 finance-news 东财", 5)
	for _, sk := range matched {
		if ClassifyPath(sk.Path) == KindExtension {
			t.Fatalf("extension skill should not match: %s", sk.Name)
		}
	}
}

func TestSummarizeBodyNotDescription(t *testing.T) {
	t.Parallel()
	sk := Skill{
		Name:        "demo",
		Description: "trigger words",
		Body:        "# Steps\n\n1. do thing",
		Path:        "skills/playbooks/demo/SKILL.md",
	}
	s := Summarize(sk, []string{"skills"}, PolicyFromConfig(nil))
	if s.Body == s.Description {
		t.Fatal("body should differ from description")
	}
	if s.Kind != KindPlaybook {
		t.Fatalf("kind=%s", s.Kind)
	}
	if s.Provenance != ProvenanceCore {
		t.Fatalf("provenance=%s", s.Provenance)
	}
	if !s.InjectInChat {
		t.Fatal("playbook should inject")
	}
}

func TestPolicyDisablesSkill(t *testing.T) {
	t.Parallel()
	cfg := config.SkillsConfig{Disabled: []string{"stock-analysis"}}
	p := PolicyFromConfig(&cfg)
	if p.Enabled("stock-analysis", KindPlaybook) {
		t.Fatal("expected disabled")
	}
}
