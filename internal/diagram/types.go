package diagram

import (
	"context"
	"encoding/json"
)

const (
	KindWorkflow     = "workflow"
	KindArchitecture = "architecture"

	ViewPathPrefix = "/v1/diagrams/"
	ViewPathSuffix = "/view"

	IDRuntimeArchitecture = "architecture.runtime"
)

// Service is the callable diagram module. Callers never talk to Node/Archify.
type Service interface {
	List() []Info
	Get(id string) (Document, error)
	Render(ctx context.Context, id string) (Artifact, error)
	RenderDocument(ctx context.Context, doc Document) (Artifact, error)
}

// Info is catalog metadata for one diagram.
type Info struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Kind     string `json:"type"`
	Skill    string `json:"skill,omitempty"`
	ViewPath string `json:"view_url"`
}

// Map returns the dashboard workflow_detail.diagram object.
func (i Info) Map() map[string]any {
	out := map[string]any{
		"id":       i.ID,
		"type":     i.Kind,
		"title":    i.Title,
		"view_url": i.ViewPath,
	}
	if i.Skill != "" {
		out["skill"] = i.Skill
	}
	return out
}

// Document is one typed Archify-compatible IR document.
type Document struct {
	ID    string
	Title string
	Kind  string
	IR    json.RawMessage
}

// Artifact is a self-contained HTML diagram.
type Artifact struct {
	HTML        []byte
	SHA256      string
	ContentType string
	Source      string // "artifact" | "archify" | "builtin"
}

func viewPath(id string) string {
	return ViewPathPrefix + id + ViewPathSuffix
}

func workflowID(skill string) string {
	return "workflow." + skill
}
