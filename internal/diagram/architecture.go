package diagram

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type architectureIR struct {
	SchemaVersion int                 `json:"schema_version"`
	DiagramType   string              `json:"diagram_type"`
	Meta          architectureMeta    `json:"meta"`
	Components    []architectureNode  `json:"components"`
	Boundaries    []architectureBound `json:"boundaries,omitempty"`
	Connections   []architectureEdge  `json:"connections,omitempty"`
	Cards         []workflowCard      `json:"cards,omitempty"`
}

type architectureMeta struct {
	Title          string `json:"title"`
	Locale         string `json:"locale,omitempty"`
	QualityProfile string `json:"quality_profile"`
}

type architectureNode struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Label    string `json:"label"`
	Sublabel string `json:"sublabel,omitempty"`
	Pos      []int  `json:"pos"`
	Size     []int  `json:"size"`
}

type architectureBound struct {
	Kind  string   `json:"kind"`
	Label string   `json:"label"`
	Wraps []string `json:"wraps"`
}

type architectureEdge struct {
	ID       string `json:"id"`
	From     string `json:"from"`
	To       string `json:"to"`
	Label    string `json:"label,omitempty"`
	Variant  string `json:"variant,omitempty"`
	FromSide string `json:"fromSide,omitempty"`
	ToSide   string `json:"toSide,omitempty"`
	Route    string `json:"route,omitempty"`
	LabelDy  int    `json:"labelDy,omitempty"`
}

func compileRuntimeArchitecture(sourceDir string) (Document, error) {
	if sourceDir != "" {
		path := filepath.Join(sourceDir, IDRuntimeArchitecture+".json")
		if raw, err := os.ReadFile(path); err == nil && len(raw) > 0 {
			return Document{
				ID:    IDRuntimeArchitecture,
				Title: "GeeGooAgent Runtime",
				Kind:  KindArchitecture,
				IR:    json.RawMessage(raw),
			}, nil
		}
	}
	ir := architectureIR{
		SchemaVersion: 1,
		DiagramType:   KindArchitecture,
		Meta: architectureMeta{
			Title:          "GeeGooAgent Runtime",
			Locale:         "zh-CN",
			QualityProfile: "standard",
		},
		Components: []architectureNode{
			{ID: "user", Type: "external", Label: "Operator", Sublabel: "Chat / Cockpit", Pos: []int{40, 220}, Size: []int{130, 64}},
			{ID: "chat_ui", Type: "frontend", Label: "Agent Chat", Sublabel: "trading_operation", Pos: []int{220, 120}, Size: []int{150, 64}},
			{ID: "cockpit", Type: "frontend", Label: "Agent Mode", Sublabel: "Workflow 页", Pos: []int{220, 320}, Size: []int{150, 64}},
			{ID: "runtime", Type: "backend", Label: "GeeGooAgent", Sublabel: "runtimeapi :3400", Pos: []int{430, 220}, Size: []int{150, 64}},
			{ID: "workflow_engine", Type: "backend", Label: "Workflow", Sublabel: "L5 + Chat flow", Pos: []int{650, 120}, Size: []int{150, 64}},
			{ID: "tools", Type: "messagebus", Label: "Tools", Sublabel: "Registry / MCP", Pos: []int{650, 220}, Size: []int{150, 64}},
			{ID: "memory", Type: "database", Label: "Memory", Sublabel: "session / facts", Pos: []int{650, 320}, Size: []int{150, 64}},
			{ID: "mcp", Type: "cloud", Label: "GeeGooBot", Sublabel: "MCP :3120", Pos: []int{860, 120}, Size: []int{140, 60}},
			{ID: "data", Type: "cloud", Label: "GeeGooData", Sublabel: "quotes / news", Pos: []int{860, 220}, Size: []int{140, 60}},
			{ID: "signal", Type: "cloud", Label: "GeeGooSignal", Sublabel: "catalog / analyze", Pos: []int{860, 320}, Size: []int{140, 60}},
		},
		Connections: []architectureEdge{
			{ID: "user-chat", From: "user", To: "chat_ui", Label: "chat", Variant: "emphasis"},
			{ID: "user-cockpit", From: "user", To: "cockpit", Label: "ops"},
			{ID: "chat-runtime", From: "chat_ui", To: "runtime", Label: "SSE"},
			{ID: "cockpit-runtime", From: "cockpit", To: "runtime", Label: "API"},
			{ID: "runtime-flow", From: "runtime", To: "workflow_engine", Label: "flow"},
			{ID: "runtime-tools", From: "runtime", To: "tools", Label: "ReAct"},
			{ID: "runtime-memory", From: "runtime", To: "memory"},
			{ID: "flow-tools", From: "workflow_engine", To: "tools", Variant: "dashed"},
			{ID: "tools-mcp", From: "tools", To: "mcp", Label: "MCP"},
			{ID: "tools-data", From: "tools", To: "data"},
			{ID: "tools-signal", From: "tools", To: "signal"},
		},
		Cards: []workflowCard{
			{Dot: "cyan", Title: "入口", Items: []string{"Chat 走 /v1/chat/stream", "Workflow 页读 /v1/dashboard/data 与 /v1/diagrams"}},
			{Dot: "emerald", Title: "编排", Items: []string{"L5 Runner 硬编码步骤", "Chat Workflow 在 ReAct 前拦截"}},
		},
	}
	raw, err := json.Marshal(ir)
	if err != nil {
		return Document{}, err
	}
	return Document{
		ID:    IDRuntimeArchitecture,
		Title: ir.Meta.Title,
		Kind:  KindArchitecture,
		IR:    raw,
	}, nil
}
