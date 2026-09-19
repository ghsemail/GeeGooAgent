package diagram

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestWorkflowIRCoversSkillSteps(t *testing.T) {
	for _, spec := range skills.Default().List() {
		doc, err := compileSkill(spec)
		if err != nil {
			t.Fatalf("%s: %v", spec.Name, err)
		}
		steps := collectSteps(spec)
		if len(steps) == 0 {
			t.Fatalf("%s: compiler produced a document but collectSteps is empty", spec.Name)
		}
		labels := nodeLabels(doc)
		ids := nodeIDs(doc)
		labelSet := map[string]bool{}
		idSet := map[string]bool{}
		for _, l := range labels {
			labelSet[l] = true
		}
		for _, id := range ids {
			idSet[id] = true
		}
		for _, step := range steps {
			if labelSet[step.Name] {
				continue
			}
			if idSet[Slug(step.Name)] {
				continue
			}
			t.Fatalf("%s: step %q missing from IR labels=%s ids=%s", spec.Name, step.Name, strings.Join(labels, ","), strings.Join(ids, ","))
		}
	}
}

func TestCatalogIDsStable(t *testing.T) {
	svc := New(".")
	seen := map[string]bool{}
	for _, info := range svc.List() {
		if info.ID == "" || info.ViewPath == "" {
			t.Fatalf("incomplete catalog row: %+v", info)
		}
		if seen[info.ID] {
			t.Fatalf("duplicate id %s", info.ID)
		}
		seen[info.ID] = true
	}
	if !seen["workflow.premarket_stock"] || !seen[IDRuntimeArchitecture] {
		t.Fatalf("missing required catalog ids: %v", seen)
	}
}
