package tools

import (
	"fmt"
	"sort"
	"strings"
)

const maxDiscoverResults = 20

// SearchTools finds deferred or all tools by keyword against name, description, and domain.
func (r *Registry) SearchTools(query string, limit int) []map[string]any {
	if r == nil {
		return nil
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if limit <= 0 {
		limit = maxDiscoverResults
	}
	type scored struct {
		score int
		item  map[string]any
	}
	var matches []scored
	for _, name := range r.ListNames() {
		t, ok := r.tools[name]
		if !ok {
			continue
		}
		def := DefinitionFromTool(t)
		score := scoreToolMatch(query, def)
		if score <= 0 {
			continue
		}
		matches = append(matches, scored{
			score: score,
			item: map[string]any{
				"name":        def.Name,
				"description": def.Description,
				"domain":      string(def.Domain),
				"toolsets":    def.Toolsets,
				"defer_load":  def.DeferLoad,
				"policy":      string(def.Policy),
			},
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].score != matches[j].score {
			return matches[i].score > matches[j].score
		}
		return matches[i].item["name"].(string) < matches[j].item["name"].(string)
	})
	if len(matches) > limit {
		matches = matches[:limit]
	}
	out := make([]map[string]any, 0, len(matches))
	for _, m := range matches {
		out = append(out, m.item)
	}
	return out
}

func scoreToolMatch(query string, def ToolDefinition) int {
	if query == "" {
		if def.DeferLoad {
			return 1
		}
		return 0
	}
	score := 0
	lowerName := strings.ToLower(def.Name)
	if lowerName == query {
		score += 100
	} else if strings.Contains(lowerName, query) {
		score += 40
	}
	if strings.Contains(strings.ToLower(def.Description), query) {
		score += 20
	}
	if strings.Contains(string(def.Domain), query) {
		score += 10
	}
	for _, ts := range def.Toolsets {
		if strings.Contains(ts, query) {
			score += 15
		}
	}
	return score
}

// RegisterDiscoverTools registers discover_tools and activate_toolset meta tools.
func RegisterDiscoverTools(r *Registry) {
	if r == nil {
		return
	}
	readOnly := true
	concurrencySafe := true
	r.Register(Tool{
		Name:        "discover_tools",
		Description: "按关键词搜索延迟加载的工具（名称/描述/域）。需要 bot、报告或运营类工具时先搜索，再 activate_toolset。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "关键词，如 dca、report、signal",
				},
			},
			"required": []any{"query"},
		},
		Spec: ToolSpec{
			ReadOnly:        &readOnly,
			ConcurrencySafe: &concurrencySafe,
		},
		Handle: func(ctx Context, args map[string]any) Result {
			query := strArg(args, "query", "")
			if strings.TrimSpace(query) == "" {
				return Result{Status: StatusError, Summary: "query is required", ExitCode: 1}
			}
			matches := r.SearchTools(query, maxDiscoverResults)
			return Result{
				Status:  StatusOK,
				Summary: fmt.Sprintf("found %d tool(s) for %q", len(matches), query),
				Data:    map[string]any{"query": query, "tools": matches},
			}
		},
	})
	r.Register(Tool{
		Name:        "activate_toolset",
		Description: "为当前会话临时展开一个 toolset（如 trading_bot、report_write）。展开后相关工具会出现在后续轮次的 schema 中。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"toolset": map[string]any{
					"type":        "string",
					"description": "toolset id，如 trading_bot、strategy、report_write",
				},
			},
			"required": []any{"toolset"},
		},
		Spec: ToolSpec{
			ReadOnly:        &readOnly,
			ConcurrencySafe: &concurrencySafe,
		},
		Handle: func(ctx Context, args map[string]any) Result {
			id := strArg(args, "toolset", "")
			if id == "" {
				id = strArg(args, "toolset_id", "")
			}
			id = strings.TrimSpace(id)
			if id == "" {
				return Result{Status: StatusError, Summary: "toolset is required", ExitCode: 1}
			}
			ts, ok := ToolsetByID(id)
			if !ok {
				return Result{
					Status: StatusError, Summary: "unknown toolset: " + id, ExitCode: 1,
					Data: map[string]any{"toolset": id},
				}
			}
			if ctx.ActiveToolsets == nil {
				return Result{
					Status: StatusError, Summary: "session toolsets not available", ExitCode: 1,
				}
			}
			for _, existing := range *ctx.ActiveToolsets {
				if existing == id {
					return Result{
						Status:  StatusOK,
						Summary: "toolset already active: " + id,
						Data:    map[string]any{"toolset": id, "tool_count": len(ts.Names())},
					}
				}
			}
			*ctx.ActiveToolsets = append(*ctx.ActiveToolsets, id)
			return Result{
				Status:  StatusOK,
				Summary: "activated toolset: " + id,
				Data:    map[string]any{"toolset": id, "tool_count": len(ts.Names()), "label": ts.Label},
			}
		},
	})
}
