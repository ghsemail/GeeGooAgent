package chat

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// FormatGenerateStrategyArchiveMessage is the canonical Dock utterance for this workflow.
func FormatGenerateStrategyArchiveMessage(strategyName string) string {
	name := strings.TrimSpace(strategyName)
	if name == "" {
		return "帮我生成策略档案"
	}
	return fmt.Sprintf("帮我生成 %s 的策略档案", name)
}

func strategyArchiveSearchQuery(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "策略档案"
	}
	return label + " 策略档案"
}

func isStrategyArchiveContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	if strings.Contains(content, "doc_type: strategy_agent_cognition") {
		return true
	}
	if strings.Contains(content, "一句话定位") {
		if strings.Contains(content, "策略档案") || strings.Contains(content, "Agent 策略档案") {
			return true
		}
		if strings.Contains(content, "Agent 策略认知") {
			return true
		}
	}
	return false
}

func filterStrategyArchiveHits(hits []any) []any {
	out := make([]any, 0, len(hits))
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if isStrategyArchiveContent(hitFieldString(row, "content")) {
			out = append(out, row)
		}
	}
	return out
}

func (r *Runner) searchStrategyArchiveHits(
	ctx context.Context,
	label string,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) []any {
	query := strategyArchiveSearchQuery(label)
	folders := append(tools.ExpandKnowledgeFolderFilter(tools.StrategyArchiveFolder), "")
	for _, folder := range folders {
		args := map[string]any{"query": query}
		if folder != "" {
			args["folder_path"] = folder
		}
		res := r.runTool(ctx, toolCtx, "search_knowledge", args, recordTool)
		if res.Status != tools.StatusOK {
			continue
		}
		hits, ok := res.Data["hits"].([]any)
		if !ok || len(hits) == 0 {
			continue
		}
		if filtered := filterStrategyArchiveHits(hits); len(filtered) > 0 {
			return filtered
		}
	}
	return nil
}
