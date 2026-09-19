package diagram

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

func TestServiceGetAndRender(t *testing.T) {
	svc := New(".")
	doc, err := svc.Get("workflow.multi_strategy_compare")
	if err != nil {
		t.Fatal(err)
	}
	if doc.Title == "" {
		t.Fatal("empty title")
	}
	art, err := svc.Render(t.Context(), doc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if art.ContentType == "" || len(art.HTML) == 0 {
		t.Fatalf("bad artifact: %+v", art)
	}
}

func TestServiceUnknownID(t *testing.T) {
	_, err := New(".").Get("workflow.not_a_skill")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLookupAndAttach(t *testing.T) {
	info, ok := LookupSkill("premarket_market")
	if !ok || info.ViewPath != "/v1/diagrams/workflow.premarket_market/view" {
		t.Fatalf("lookup: %+v ok=%v", info, ok)
	}
	items := []map[string]any{
		{"name": "premarket_market", "workflow_detail": map[string]any{"name": "premarket_market"}},
		{"name": "bot-manager"},
	}
	Attach(items)
	detail := items[0]["workflow_detail"].(map[string]any)
	diagram, _ := detail["diagram"].(map[string]any)
	if diagram["id"] != "workflow.premarket_market" {
		t.Fatalf("diagram=%v", diagram)
	}
	if items[1]["workflow_detail"] != nil {
		t.Fatal("playbook should stay untouched")
	}
}

func TestWriteAndLoadArtifact(t *testing.T) {
	dir := t.TempDir()
	spec, _ := skills.Default().Get("param_tune")
	doc, err := compileSkill(spec)
	if err != nil {
		t.Fatal(err)
	}
	art, err := BuiltinRenderer{}.Render(t.Context(), doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := WriteArtifact(dir, doc.ID, art); err != nil {
		t.Fatal(err)
	}
	got, ok := loadArtifact(dir, doc.ID)
	if !ok || got.SHA256 != art.SHA256 {
		t.Fatalf("cache miss: ok=%v got=%s want=%s", ok, got.SHA256, art.SHA256)
	}
	if _, err := os.Stat(filepath.Join(dir, ArtifactDir, "workflow.param_tune.html")); err != nil {
		t.Fatal(err)
	}
}

func TestChecks(t *testing.T) {
	rows := Checks(".")
	if len(rows) < 2 {
		t.Fatalf("checks=%d", len(rows))
	}
}
