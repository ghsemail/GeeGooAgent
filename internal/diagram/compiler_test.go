package diagram

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestCompileAllWorkflowSkills(t *testing.T) {
	for _, spec := range skills.Default().List() {
		doc, err := compileSkill(spec)
		if err != nil {
			t.Fatalf("%s: %v", spec.Name, err)
		}
		if doc.Kind != KindWorkflow {
			t.Fatalf("%s kind=%s", spec.Name, doc.Kind)
		}
		var ir workflowIR
		if err := json.Unmarshal(doc.IR, &ir); err != nil {
			t.Fatalf("%s ir: %v", spec.Name, err)
		}
		if ir.SchemaVersion != 2 || ir.DiagramType != KindWorkflow {
			t.Fatalf("%s unexpected header %+v", spec.Name, ir)
		}
		if len(ir.Nodes) == 0 || len(ir.Lanes) == 0 {
			t.Fatalf("%s missing nodes/lanes", spec.Name)
		}
		if ir.Meta.QualityProfile != "standard" {
			t.Fatalf("%s quality=%s", spec.Name, ir.Meta.QualityProfile)
		}
	}
}

func TestCompileRuntimeArchitecture(t *testing.T) {
	doc, err := compileRuntimeArchitecture("")
	if err != nil {
		t.Fatal(err)
	}
	var ir architectureIR
	if err := json.Unmarshal(doc.IR, &ir); err != nil {
		t.Fatal(err)
	}
	if len(ir.Components) < 6 || len(ir.Connections) < 4 {
		t.Fatalf("sparse architecture: %+v", ir)
	}
}

func TestBuiltinRenderWorkflow(t *testing.T) {
	spec, ok := skills.Default().Get("strategy_dev")
	if !ok {
		t.Fatal("strategy_dev missing")
	}
	doc, err := compileSkill(spec)
	if err != nil {
		t.Fatal(err)
	}
	art, err := BuiltinRenderer{}.Render(t.Context(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if len(art.HTML) < 200 || art.SHA256 == "" {
		t.Fatalf("short artifact: %d", len(art.HTML))
	}
}
