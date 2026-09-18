package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func newGenerateStrategyCognitionFlow(userText string) *Flow {
	now := time.Now().UTC()
	query := extractStrategyQuery(userText)
	return &Flow{
		RunID:         newRunID(),
		Template:      SkillGenerateStrategyCognition,
		Status:        StatusRunning,
		WorkflowStep:  StepGenerateCognition,
		Phase:         PhaseCognitionPick,
		StrategyQuery: query,
		TriggerText:   strings.TrimSpace(userText),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (r *Runner) advanceGenerateStrategyCognition(
	ctx context.Context,
	_ *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	switch flow.Phase {
	case PhaseCognitionPick:
		if strings.TrimSpace(flow.StrategyQuery) == "" {
			return terminalError("请说明要生成认知的策略名称，例如：生成策略认知 Macd4H")
		}
		flow.Phase = PhaseCognitionReadCatalog
		flow.touch()
		return nil
	case PhaseCognitionReadCatalog:
		match, err := resolveStrategyCatalog(ctx, flow.StrategyQuery, toolCtx, r.runToolCall(recordTool))
		if err != nil {
			return err
		}
		flow.CatalogType = match.Type
		flow.CatalogLabel = match.Label
		flow.CatalogRaw = match.Raw
		if strings.TrimSpace(flow.StrategyQuery) == "" {
			flow.StrategyQuery = match.Label
		}
		flow.Phase = PhaseCognitionCompose
		flow.touch()
		return nil
	case PhaseCognitionWebResearch:
		fallthrough
	case PhaseCognitionCompose:
		return r.phaseCognitionCompose(ctx, flow, toolCtx, recordTool)
	case PhaseCognitionSaveKB:
		return r.phaseCognitionSaveKB(ctx, flow, toolCtx, recordTool)
	case PhaseCognitionVerifyKB:
		return r.phaseCognitionVerifyKB(ctx, flow, toolCtx, recordTool)
	case PhaseSummarize, PhaseDone:
		return nil
	default:
		flow.Phase = PhaseCognitionPick
		return nil
	}
}

func (r *Runner) phaseCognitionSaveKB(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	content := strings.TrimSpace(flow.KBDraft)
	if content == "" {
		return terminalError("缺少知识库正文")
	}
	name := strategyDisplayLabel(flow)
	res := r.runTool(ctx, toolCtx, "save_strategy_knowledge", map[string]any{
		"strategy_name": name,
		"content":       content,
		"folder_path":   defaultCognitionFolder,
	}, recordTool)
	if res.Status != tools.StatusOK {
		return fmt.Errorf("save_strategy_knowledge 失败：%s", res.Summary)
	}
	flow.KnowledgeID = strings.TrimSpace(fmt.Sprint(res.Data["knowledge_id"]))
	flow.KnowledgeTitle = strings.TrimSpace(fmt.Sprint(res.Data["title"]))
	flow.Phase = PhaseCognitionVerifyKB
	flow.touch()
	return nil
}

func (r *Runner) phaseCognitionVerifyKB(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	label := strategyDisplayLabel(flow)
	knowledgeID := strings.TrimSpace(flow.KnowledgeID)
	deadline := time.Now().Add(cognitionParseWait * time.Second)
	attempt := 0
	for time.Now().Before(deadline) {
		if knowledgeID != "" {
			res := r.runTool(ctx, toolCtx, "get_knowledge", map[string]any{"knowledge_id": knowledgeID}, recordTool)
			if res.Status == tools.StatusOK {
				content := strings.TrimSpace(fmt.Sprint(res.Data["content"]))
				if isAgentCognitionContent(content) {
					flow.VerifySnippet = truncate(content, 240)
					flow.Phase = PhaseSummarize
					flow.touch()
					return nil
				}
			}
		}
		args := map[string]any{"query": label + " Agent 策略认知"}
		if attempt < 15 {
			args["folder_path"] = defaultCognitionFolder
		}
		res := r.runTool(ctx, toolCtx, "search_knowledge", args, recordTool)
		if res.Status == tools.StatusOK {
			if hits, ok := res.Data["hits"].([]any); ok {
				if snippet := firstCognitionHitPreview(hits); snippet != "" {
					flow.VerifySnippet = snippet
					flow.Phase = PhaseSummarize
					flow.touch()
					return nil
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		attempt++
		time.Sleep(2 * time.Second)
	}
	return terminalError("知识库读回验证超时：未读到 Agent 策略认知正文，请稍后重试")
}

func isAgentCognitionContent(content string) bool {
	content = strings.TrimSpace(content)
	if content == "" {
		return false
	}
	return strings.Contains(content, "doc_type: strategy_agent_cognition") ||
		strings.Contains(content, "Agent 策略认知")
}

func firstCognitionHitPreview(hits []any) string {
	for _, raw := range hits {
		content := hitContent(raw)
		if !isAgentCognitionContent(content) {
			continue
		}
		return truncate(content, 240)
	}
	return ""
}

func firstHitPreview(hits []any) string {
	for _, raw := range hits {
		content := hitContent(raw)
		if content != "" {
			return truncate(content, 240)
		}
	}
	return ""
}

func hitContent(raw any) string {
	row, ok := raw.(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(row["content"]))
}

func filterCognitionHits(hits []any) []any {
	out := make([]any, 0, len(hits))
	for _, raw := range hits {
		if isAgentCognitionContent(hitContent(raw)) {
			out = append(out, raw)
		}
	}
	return out
}

func renderGenerateCognitionReport(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	var b strings.Builder
	fmt.Fprintf(&b, "## 生成策略认知完成 · %s\n\n", label)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 策略库 | %s（%s） |\n", label, flow.CatalogType)
	fmt.Fprintf(&b, "| 认知合成 | LLM 基于策略库 |\n")
	if flow.KnowledgeID != "" {
		fmt.Fprintf(&b, "| 知识库 | [%s](kb:%s) · %s |\n", flow.KnowledgeTitle, flow.KnowledgeID, defaultCognitionFolder)
	}
	if flow.VerifySnippet != "" {
		fmt.Fprintf(&b, "\n**读回验证片段**：\n\n> %s\n", flow.VerifySnippet)
	}
	fmt.Fprintf(&b, "\n> 策略认知已写入知识库；策略开发 workflow 将从此处读取。")
	return strings.TrimSpace(b.String())
}

func renderGenerateCognitionPartial(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	return fmt.Sprintf("## 生成策略认知（进行中）\n\n- 策略：%s\n- 阶段：%s",
		label, flow.Phase)
}

func strategyDisplayLabel(flow *Flow) string {
	if flow == nil {
		return ""
	}
	if label := strings.TrimSpace(flow.CatalogLabel); label != "" {
		return label
	}
	return strings.TrimSpace(flow.StrategyQuery)
}
