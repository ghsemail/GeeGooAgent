package diagram

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/skills"
)

type namedStep struct {
	Name string
	Tool string
	Kind string // prelude | perstock | chat
}

type workflowIR struct {
	SchemaVersion int             `json:"schema_version"`
	DiagramType   string          `json:"diagram_type"`
	Meta          workflowMeta    `json:"meta"`
	Lanes         []workflowLane  `json:"lanes"`
	Phases        []workflowPhase `json:"phases,omitempty"`
	MainPath      []string        `json:"mainPath,omitempty"`
	Nodes         []workflowNode  `json:"nodes"`
	Edges         []workflowEdge  `json:"edges"`
	Cards         []workflowCard  `json:"cards,omitempty"`
}

type workflowMeta struct {
	Title          string `json:"title"`
	Locale         string `json:"locale,omitempty"`
	QualityProfile string `json:"quality_profile"`
}

type workflowLane struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

type workflowPhase struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	FromCol int    `json:"fromCol"`
	ToCol   int    `json:"toCol"`
}

type workflowNode struct {
	ID       string  `json:"id"`
	Lane     string  `json:"lane"`
	Col      int     `json:"col"`
	Type     string  `json:"type"`
	Label    string  `json:"label"`
	Sublabel string  `json:"sublabel,omitempty"`
	Width    float64 `json:"width,omitempty"`
}

type workflowEdge struct {
	ID      string `json:"id"`
	From    string `json:"from"`
	To      string `json:"to"`
	Label   string `json:"label,omitempty"`
	Variant string `json:"variant,omitempty"`
	Role    string `json:"role,omitempty"`
}

type workflowCard struct {
	Dot   string   `json:"dot"`
	Title string   `json:"title"`
	Items []string `json:"items"`
}

func compileSkill(spec skills.Spec) (Document, error) {
	steps := collectSteps(spec)
	if len(steps) == 0 {
		return Document{}, fmt.Errorf("diagram: skill %s has no steps or phases", spec.Name)
	}
	title := skillTitle(spec)
	ir := workflowIR{
		SchemaVersion: 2,
		DiagramType:   KindWorkflow,
		Meta: workflowMeta{
			Title:          title,
			Locale:         "zh-CN",
			QualityProfile: "standard",
		},
	}

	lanes := laneSet(steps)
	for _, lane := range lanes {
		ir.Lanes = append(ir.Lanes, workflowLane{ID: lane.id, Label: lane.label})
	}

	used := map[string]int{}
	mainPath := make([]string, 0, 6)
	mainOpen := true
	indexInKind := map[string]int{}
	var prev string
	for i, step := range steps {
		id := uniqueID(used, step.Name)
		row := indexInKind[step.Kind]
		indexInKind[step.Kind] = row + 1
		col := row % 6
		lane := laneID(step.Kind, row/6)
		node := workflowNode{
			ID:    id,
			Lane:  lane,
			Col:   col,
			Type:  nodeType(step),
			Label: step.Name,
			Width: nodeWidth(step.Name),
		}
		if step.Tool != "" && step.Tool != "(chat workflow)" {
			node.Sublabel = truncateLabel(step.Tool, 22)
		}
		ir.Nodes = append(ir.Nodes, node)
		if prev != "" {
			edge := workflowEdge{
				ID:   uniqueID(used, "e_"+prev+"_"+id),
				From: prev,
				To:   id,
				Role: "main",
			}
			if col == 0 && row > 0 {
				edge.Role = "return"
				edge.Variant = "dashed"
			} else if i > 0 && step.Kind != steps[i-1].Kind {
				edge.Variant = "dashed"
				edge.Role = "branch"
			}
			ir.Edges = append(ir.Edges, edge)
		}
		if mainOpen && col == len(mainPath) && len(mainPath) < 6 {
			mainPath = append(mainPath, id)
		} else if len(mainPath) > 0 {
			mainOpen = false
		}
		prev = id
	}
	if len(mainPath) >= 2 {
		ir.MainPath = mainPath
	}
	ir.Phases = phasesFor(steps)
	ir.Cards = cardsFor(spec)

	raw, err := json.Marshal(ir)
	if err != nil {
		return Document{}, err
	}
	return Document{
		ID:    workflowID(spec.Name),
		Title: title,
		Kind:  KindWorkflow,
		IR:    raw,
	}, nil
}

func collectSteps(spec skills.Spec) []namedStep {
	detail := skills.BuildWorkflowDetail("", spec, nil, "")
	out := make([]namedStep, 0)
	out = append(out, readDetailSteps(detail["phase_a_steps"], "prelude")...)
	out = append(out, readDetailSteps(detail["phase_b_steps"], "perstock")...)
	if len(out) == 0 {
		if phases, _ := detail["chat_phases"].([]string); len(phases) > 0 {
			for _, phase := range phases {
				out = append(out, namedStep{Name: phase, Tool: "(chat workflow)", Kind: "chat"})
			}
		}
	}
	return out
}

func readDetailSteps(raw any, kind string) []namedStep {
	rows, ok := raw.([]map[string]any)
	if !ok {
		return nil
	}
	out := make([]namedStep, 0, len(rows))
	for _, row := range rows {
		name, _ := row["name"].(string)
		tool, _ := row["tool"].(string)
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		out = append(out, namedStep{Name: name, Tool: strings.TrimSpace(tool), Kind: kind})
	}
	return out
}

func skillTitle(spec skills.Spec) string {
	desc := strings.TrimSpace(spec.Description)
	if strings.HasPrefix(desc, "【") {
		if end := strings.Index(desc, "】"); end > 0 {
			return desc[len("【"):end]
		}
	}
	if i := strings.Index(desc, "："); i > 0 && i < 24 {
		return desc[:i]
	}
	if i := strings.Index(desc, ":"); i > 0 && i < 24 {
		return desc[:i]
	}
	if spec.Name != "" {
		return spec.Name
	}
	return "Workflow"
}

type laneDef struct {
	id    string
	label string
}

func laneSet(steps []namedStep) []laneDef {
	seen := map[string]bool{}
	count := map[string]int{}
	out := make([]laneDef, 0, 4)
	for _, step := range steps {
		row := count[step.Kind]
		count[step.Kind] = row + 1
		id := laneID(step.Kind, row/6)
		if seen[id] {
			continue
		}
		seen[id] = true
		label := laneLabel(step.Kind)
		if row/6 > 0 {
			label = label + " · " + itoa(row/6+1)
		}
		out = append(out, laneDef{id: id, label: label})
	}
	return out
}

func laneID(kind string, row int) string {
	base := "prelude"
	switch kind {
	case "perstock":
		base = "perstock"
	case "chat":
		base = "chat"
	}
	if row <= 0 {
		return base
	}
	return base + "_r" + itoa(row+1)
}

func laneLabel(kind string) string {
	switch kind {
	case "perstock":
		return "Phase B · 逐股"
	case "chat":
		return "Chat Workflow"
	default:
		return "Phase A · Prelude"
	}
}

func nodeType(step namedStep) string {
	tool := strings.ToLower(step.Tool)
	name := strings.ToLower(step.Name)
	switch {
	case strings.Contains(tool, "news") || strings.Contains(name, "news"):
		return "external"
	case strings.Contains(tool, "save") || strings.Contains(tool, "create") || strings.Contains(name, "save") || strings.Contains(name, "create"):
		return "database"
	case strings.Contains(tool, "kb") || strings.Contains(name, "kb") || strings.Contains(name, "knowledge"):
		return "database"
	case strings.Contains(name, "summarize") || strings.Contains(name, "complete"):
		return "backend"
	case step.Kind == "chat":
		return "frontend"
	default:
		return "backend"
	}
}

func nodeWidth(label string) float64 {
	w := float64(len([]rune(label))*8 + 28)
	if w < 140 {
		return 140
	}
	if w > 220 {
		return 220
	}
	return w
}

func truncateLabel(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}

func phasesFor(steps []namedStep) []workflowPhase {
	if len(steps) == 0 {
		return nil
	}
	out := []workflowPhase{}
	start := 0
	kind := steps[0].Kind
	for i := 1; i <= len(steps); i++ {
		if i == len(steps) || steps[i].Kind != kind {
			from := start % 6
			to := (i - 1) % 6
			if to < from {
				to = 5
			}
			out = append(out, workflowPhase{
				ID:      Slug(kind + "_phase"),
				Label:   laneLabel(kind),
				FromCol: from,
				ToCol:   to,
			})
			if i < len(steps) {
				start = i
				kind = steps[i].Kind
			}
		}
	}
	return out
}

func cardsFor(spec skills.Spec) []workflowCard {
	cards := []workflowCard{{
		Dot:   "cyan",
		Title: "Skill",
		Items: []string{spec.Name, strings.TrimSpace(spec.Description)},
	}}
	if spec.Chat != nil {
		items := append([]string{}, spec.Chat.Triggers...)
		if spec.Chat.Status != "" {
			items = append(items, "status: "+spec.Chat.Status)
		}
		if len(items) > 0 {
			cards = append(cards, workflowCard{Dot: "emerald", Title: "Chat 触发", Items: items})
		}
	}
	return cards
}

func nodeLabels(doc Document) []string {
	var raw struct {
		Nodes []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"nodes"`
		Components []struct {
			ID    string `json:"id"`
			Label string `json:"label"`
		} `json:"components"`
	}
	_ = json.Unmarshal(doc.IR, &raw)
	out := make([]string, 0, len(raw.Nodes)+len(raw.Components))
	for _, n := range raw.Nodes {
		out = append(out, n.Label)
	}
	for _, n := range raw.Components {
		out = append(out, n.Label)
	}
	return out
}

func nodeIDs(doc Document) []string {
	var raw struct {
		Nodes []struct {
			ID string `json:"id"`
		} `json:"nodes"`
		Components []struct {
			ID string `json:"id"`
		} `json:"components"`
	}
	_ = json.Unmarshal(doc.IR, &raw)
	out := make([]string, 0, len(raw.Nodes)+len(raw.Components))
	for _, n := range raw.Nodes {
		out = append(out, n.ID)
	}
	for _, n := range raw.Components {
		out = append(out, n.ID)
	}
	return out
}
