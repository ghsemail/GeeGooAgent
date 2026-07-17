package skills_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestDefaultRegistryHasPreMarket(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	spec, ok := r.Get("pre_market")
	if !ok {
		t.Fatal("pre_market not registered")
	}
	if spec.PhaseA == nil || spec.PerStock == nil {
		t.Fatal("pre_market step functions nil")
	}
	if len(spec.PhaseA()) == 0 {
		t.Fatal("pre_market phase A steps empty")
	}
}

func TestUnknownSkillErrors(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	if _, ok := r.Get("nonexistent"); ok {
		t.Fatal("expected missing for unknown skill")
	}
}

func TestListIncludesBuiltinSkills(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	names := map[string]bool{}
	for _, s := range r.List() {
		names[s.Name] = true
	}
	for _, want := range []string{"pre_market", "intraday", "post_market"} {
		if !names[want] {
			t.Fatalf("missing %s in list", want)
		}
	}
}

func TestIntradayAndPostMarketHaveSteps(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	for _, name := range []string{"intraday", "post_market"} {
		spec, ok := r.Get(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if len(spec.PerStock()) == 0 {
			t.Fatalf("%s per-stock steps empty", name)
		}
	}
}
