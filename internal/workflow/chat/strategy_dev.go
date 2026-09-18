package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func newStrategyDevFlow(userText string) *Flow {
	now := time.Now().UTC()
	query := extractStrategyDevQuery(userText)
	return &Flow{
		RunID:         newRunID(),
		Template:      SkillStrategyDev,
		Status:        StatusRunning,
		WorkflowStep:  StepCognition,
		Phase:         PhaseCognitionPick,
		StrategyQuery: query,
		TriggerText:   strings.TrimSpace(userText),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func extractStrategyDevQuery(text string) string {
	repl := strings.NewReplacer(
		"策略认知", " ", "策略开发", " ", "学习一下", " ", "学习", " ", "了解", " ",
		"整理到知识库", " ", "写入知识库", " ", "存入知识库", " ", "帮我", " ", "请", " ",
		"workflow", " ", "Workflow", " ",
	)
	clean := strings.TrimSpace(repl.Replace(text))
	if clean == "" {
		return strings.TrimSpace(text)
	}
	if parts := splitStrategyQueries(clean); len(parts) >= 1 && len(parts) <= 2 {
		return strings.Join(parts, " ")
	}
	return clean
}

func (r *Runner) advanceStrategyDev(
	ctx context.Context,
	_ *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	switch flow.Phase {
	case PhaseCognitionPick:
		if strings.TrimSpace(flow.StrategyQuery) == "" {
			return terminalError("请说明要认知的策略名称，例如：策略认知 Macd4H")
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
		// Legacy persisted phase: web search removed; go straight to LLM compose.
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
	name := strings.TrimSpace(flow.CatalogLabel)
	if name == "" {
		name = flow.StrategyQuery
	}
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
	query := strings.TrimSpace(flow.CatalogLabel)
	if query == "" {
		query = flow.StrategyQuery
	}
	deadline := time.Now().Add(cognitionParseWait * time.Second)
	attempt := 0
	for time.Now().Before(deadline) {
		args := map[string]any{"query": query + " Agent 策略认知"}
		if attempt < 15 {
			args["folder_path"] = defaultCognitionFolder
		}
		res := r.runTool(ctx, toolCtx, "search_knowledge", args, recordTool)
		if res.Status == tools.StatusOK {
			if hits, ok := res.Data["hits"].([]any); ok && len(hits) > 0 {
				flow.VerifySnippet = firstHitPreview(hits)
				flow.Phase = PhaseSummarize
				flow.touch()
				return nil
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		attempt++
		time.Sleep(2 * time.Second)
	}
	return terminalError("知识库读回验证超时：search_knowledge 未命中，请稍后重试")
}

func firstHitPreview(hits []any) string {
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content := strings.TrimSpace(fmt.Sprint(row["content"]))
		if content != "" {
			return truncate(content, 240)
		}
	}
	return ""
}

func renderCognitionReport(flow *Flow) string {
	label := strings.TrimSpace(flow.CatalogLabel)
	if label == "" {
		label = flow.StrategyQuery
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## 策略认知完成 · %s\n\n", label)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 策略库 | %s（%s） |\n", label, flow.CatalogType)
	fmt.Fprintf(&b, "| 认知合成 | LLM 基于策略库（无 Web 搜索） |\n")
	if flow.KnowledgeID != "" {
		fmt.Fprintf(&b, "| 知识库 | [%s](kb:%s) · %s |\n", flow.KnowledgeTitle, flow.KnowledgeID, defaultCognitionFolder)
	}
	if flow.VerifySnippet != "" {
		fmt.Fprintf(&b, "\n**读回验证片段**：\n\n> %s\n", flow.VerifySnippet)
	}
	fmt.Fprintf(&b, "\n> Step 1「策略认知」已完成；文档含 Agent 注入指引与参数说明。后续 Step 2+ 将在同一 strategy_dev workflow 中扩展。")
	return strings.TrimSpace(b.String())
}

func renderCognitionPartial(flow *Flow) string {
	label := strings.TrimSpace(flow.CatalogLabel)
	if label == "" {
		label = flow.StrategyQuery
	}
	return fmt.Sprintf("## 策略开发 · 策略认知（进行中）\n\n- 策略：%s\n- 阶段：%s\n- Step：%s",
		label, flow.Phase, flow.WorkflowStep)
}
