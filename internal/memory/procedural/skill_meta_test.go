package procedural

import "testing"

func TestClassifyPath(t *testing.T) {
	t.Parallel()
	cases := []struct {
		path string
		want Kind
	}{
		{`skills/playbooks/stock-analysis/SKILL.md`, KindPlaybook},
		{`skills/pre_market/SKILL.md`, KindWorkflow},
		{`skills/bundled/finance-news/SKILL.md`, KindBundled},
		{`/home/ubuntu/.geegoo/skills/my-skill/SKILL.md`, KindUser},
	}
	for _, tc := range cases {
		if got := ClassifyPath(tc.path); got != tc.want {
			t.Fatalf("%s: got %q want %q", tc.path, got, tc.want)
		}
	}
}

func TestMatchSkipsBundled(t *testing.T) {
	t.Parallel()
	l := NewLoader("../../../skills")
	matched := l.Match("财经新闻 finance-news 东财", 5)
	for _, sk := range matched {
		if ClassifyPath(sk.Path) == KindBundled {
			t.Fatalf("bundled skill should not match: %s", sk.Name)
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
	s := Summarize(sk, []string{"skills"})
	if s.Body == s.Description {
		t.Fatal("body should differ from description")
	}
	if s.Kind != KindPlaybook {
		t.Fatalf("kind=%s", s.Kind)
	}
	if !s.InjectInChat {
		t.Fatal("playbook should inject")
	}
}
