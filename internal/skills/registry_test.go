package skills_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestDefaultRegistryHasPreMarketSkills(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	for _, name := range []string{"pre_market", "pre_market_stock"} {
		spec, ok := r.Get(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if spec.PhaseA == nil {
			t.Fatalf("%s phase A nil", name)
		}
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
	for _, want := range []string{"pre_market", "pre_market_stock", "intraday", "post_market"} {
		if !names[want] {
			t.Fatalf("missing %s in list", want)
		}
	}
}

func TestIntradayAndPostMarketHaveSteps(t *testing.T) {
	t.Parallel()
	r := skills.Default()
	for _, name := range []string{"intraday", "post_market", "pre_market_stock"} {
		spec, ok := r.Get(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if len(spec.PerStock()) == 0 {
			t.Fatalf("%s per-stock steps empty", name)
		}
	}
}
