package runtimeapi

import (
	"net/http"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/chatprompt"
	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
)

func (h *Handler) registerMemoryPlanRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/memory/plan", h.memoryPlanGet)
}

func (h *Handler) memoryPlanGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]any{
		"style":    "soft_guidance",
		"sections": buildPlanViewSections(),
	})
}

func buildPlanViewSections() []map[string]any {
	type section struct {
		id, title, markdown, source string
	}
	raw := []section{
		{"overview", "概览", chatprompt.PlanOverviewMarkdown(), "internal/chatprompt/plan_view.go"},
		{"routing", "Tool 路由", chatprompt.ToolRouting(), "internal/chatprompt/tool_routing.go"},
		{"classify", "Classify Prompt", cognition.ClassifyPromptCore(), "internal/cognition/llm_planner.go"},
		{"domains", "Domain → Playbook", domaincatalog.PlanCatalogMarkdown(), "internal/domaincatalog/catalog.go"},
		{"steps", "Plan Steps", agent.PlanStepsMarkdown(), "internal/agent/loop_turnplan.go"},
		{"fragment", "Plan Fragment", agent.TurnPlanFragmentTemplateDoc(), "internal/agent/loop_turnplan.go"},
	}
	out := make([]map[string]any, 0, len(raw))
	for _, s := range raw {
		out = append(out, map[string]any{
			"id":       s.id,
			"title":    s.title,
			"markdown": s.markdown,
			"source":   s.source,
		})
	}
	return out
}
