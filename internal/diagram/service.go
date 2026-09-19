package diagram

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

// Default is the in-process catalog + renderer.
type Default struct {
	Root     string
	Skills   *skills.Registry
	Renderer Renderer
	Archify  *ArchifyCLI
}

// New returns a Service bound to projectRoot.
func New(projectRoot string) *Default {
	root := FindRepoRoot(strings.TrimSpace(projectRoot))
	s := &Default{
		Root:     root,
		Skills:   skills.Default(),
		Renderer: BuiltinRenderer{},
		Archify:  &ArchifyCLI{Root: root},
	}
	return s
}

func (s *Default) registry() *skills.Registry {
	if s != nil && s.Skills != nil {
		return s.Skills
	}
	return skills.Default()
}

func (s *Default) root() string {
	if s != nil && strings.TrimSpace(s.Root) != "" {
		return s.Root
	}
	return "."
}

func (s *Default) renderer() Renderer {
	if s != nil && s.Renderer != nil {
		return s.Renderer
	}
	return BuiltinRenderer{}
}

// List returns catalog entries for every registered workflow skill plus the runtime map.
func (s *Default) List() []Info {
	out := make([]Info, 0, 12)
	for _, spec := range s.registry().List() {
		out = append(out, infoForSpec(spec))
	}
	out = append(out, Info{
		ID:       IDRuntimeArchitecture,
		Title:    "GeeGooAgent Runtime",
		Kind:     KindArchitecture,
		ViewPath: viewPath(IDRuntimeArchitecture),
	})
	return out
}

func infoForSpec(spec skills.Spec) Info {
	return Info{
		ID:       workflowID(spec.Name),
		Title:    skillTitle(spec),
		Kind:     KindWorkflow,
		Skill:    spec.Name,
		ViewPath: viewPath(workflowID(spec.Name)),
	}
}

// LookupSkill returns catalog info for a workflow skill name.
func LookupSkill(name string) (Info, bool) {
	spec, ok := skills.Default().Get(strings.TrimSpace(name))
	if !ok {
		return Info{}, false
	}
	return infoForSpec(spec), true
}

// Get compiles (or loads an override) the IR for id.
func (s *Default) Get(id string) (Document, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Document{}, fmt.Errorf("diagram: empty id")
	}
	if doc, ok := loadOverride(s.root(), id); ok {
		return doc, nil
	}
	if id == IDRuntimeArchitecture {
		return compileRuntimeArchitecture(filepath.Join(s.root(), "internal", "diagram", "sources"))
	}
	skill := strings.TrimPrefix(id, "workflow.")
	spec, ok := s.registry().Get(skill)
	if !ok {
		return Document{}, fmt.Errorf("diagram: unknown id %s", id)
	}
	return compileSkill(spec)
}

func loadOverride(root, id string) (Document, bool) {
	path := filepath.Join(root, "internal", "diagram", "sources", safeFileName(id)+".json")
	raw, err := os.ReadFile(path)
	if err != nil || len(raw) == 0 {
		return Document{}, false
	}
	kind := KindWorkflow
	title := id
	if strings.HasPrefix(id, "architecture.") {
		kind = KindArchitecture
	}
	if spec, ok := skills.Default().Get(strings.TrimPrefix(id, "workflow.")); ok {
		title = skillTitle(spec)
	}
	if id == IDRuntimeArchitecture {
		title = "GeeGooAgent Runtime"
	}
	return Document{ID: id, Title: title, Kind: kind, IR: raw}, true
}

// Render returns the cached artifact, or live-renders if the cache is missing.
func (s *Default) Render(ctx context.Context, id string) (Artifact, error) {
	if art, ok := loadArtifact(s.root(), id); ok {
		return art, nil
	}
	doc, err := s.Get(id)
	if err != nil {
		return Artifact{}, err
	}
	return s.RenderDocument(ctx, doc)
}

// RenderDocument prefers Archify when available, otherwise the builtin renderer.
func (s *Default) RenderDocument(ctx context.Context, doc Document) (Artifact, error) {
	if s != nil && s.Archify != nil && s.Archify.Available() {
		art, err := s.Archify.Render(ctx, doc)
		if err == nil {
			return art, nil
		}
		if os.Getenv("DIAGRAM_REQUIRE_ARCHIFY") == "1" {
			return Artifact{}, err
		}
	}
	return s.renderer().Render(ctx, doc)
}

// Attach writes diagram metadata into each item's workflow_detail (if present).
func Attach(items []map[string]any) {
	if len(items) == 0 {
		return
	}
	for i, item := range items {
		detail, ok := item["workflow_detail"].(map[string]any)
		if !ok {
			continue
		}
		name, _ := item["name"].(string)
		info, found := LookupSkill(name)
		if !found {
			continue
		}
		detail["diagram"] = info.Map()
		item["workflow_detail"] = detail
		items[i] = item
	}
}
