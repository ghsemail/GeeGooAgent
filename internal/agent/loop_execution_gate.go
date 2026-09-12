package agent

import (
	"context"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/domaincatalog"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func toolsFromRuntimeRecords(records []runtime.StepRecord) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, rec := range records {
		if rec.Kind != "tool" {
			continue
		}
		name := strings.TrimSpace(rec.ToolName)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	return out
}

func unionToolNames(a, b []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, name := range list {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

func (l *Loop) tryExecutionProfileRetry(
	ctx context.Context,
	session *runtime.Session,
	messages *[]llm.Message,
	turnPlan cognition.TurnPlan,
	records []runtime.StepRecord,
	retriesLeft *int,
) bool {
	if l == nil || session == nil || retriesLeft == nil || *retriesLeft <= 0 {
		return false
	}
	profileID := domaincatalog.ProbeExecutionProfile(
		domaincatalog.Domain(turnPlan.Domain),
		turnPlan.Act,
		lastUserText(session),
	)
	session.LastExecutionProfile = profileID
	if profileID == "" {
		return false
	}
	judged := toolsFromRuntimeRecords(records)
	sessionTools := unionToolNames(session.PriorSessionTools, judged)
	ok, reason := domaincatalog.VerifyExecutionProfile(profileID, judged, sessionTools)
	if ok {
		return false
	}
	*retriesLeft--
	l.emit("execution_retry", map[string]any{
		"profile": profileID, "reason": reason, "remaining": *retriesLeft,
	})
	hint := domaincatalog.ProfileExecutionHint(profileID)
	if strings.TrimSpace(hint) == "" {
		hint = reason
	}
	session.AppendMessage(llm.Message{
		Role:    llm.RoleUser,
		Content: "[执行契约] " + hint + " 请补调必要工具后再回答。",
	})
	*messages = session.LLMMessages()
	return true
}

func lastUserText(session *runtime.Session) string {
	if session == nil {
		return ""
	}
	msgs := session.LLMMessages()
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role != llm.RoleUser {
			continue
		}
		return strings.TrimSpace(msgs[i].Content)
	}
	return ""
}

func applyTurnToolSchemas(base []llm.ToolSchema, turnPlan cognition.TurnPlan) []llm.ToolSchema {
	out := cognition.FilterSchemas(base, turnPlan)
	profileID := domaincatalog.ExecutionProfileFor(domaincatalog.Domain(turnPlan.Domain), turnPlan.Act)
	return filterExecutionProfileSchemas(out, profileID)
}

func filterExecutionProfileSchemas(schemas []llm.ToolSchema, profileID string) []llm.ToolSchema {
	switch profileID {
	case domaincatalog.ProfileStockPriceViaMCP, domaincatalog.ProfileStockTechnicalFull:
		return filterOutToolSchema(schemas, "get_current_price")
	case domaincatalog.ProfileStockPriceSnapshot:
		return filterOutToolSchema(schemas, "get_mcp_analysis")
	default:
		return schemas
	}
}

func filterOutToolSchema(schemas []llm.ToolSchema, name string) []llm.ToolSchema {
	out := make([]llm.ToolSchema, 0, len(schemas))
	for _, s := range schemas {
		if s.Name == name {
			continue
		}
		out = append(out, s)
	}
	return out
}

// filterPriceShortcutSchemas is deprecated; use filterExecutionProfileSchemas.
func filterPriceShortcutSchemas(schemas []llm.ToolSchema, profileID string) []llm.ToolSchema {
	return filterExecutionProfileSchemas(schemas, profileID)
}
