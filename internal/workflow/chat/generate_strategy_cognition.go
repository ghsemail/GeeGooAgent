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
			return terminalError("请说明要生成档案的策略名称，例如：帮我生成 Macd4H 的策略档案")
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
	name := strings.TrimSpace(flow.CatalogLabel)
	if name == "" {
		name = flow.StrategyQuery
	}
	res := r.runTool(ctx, toolCtx, "save_strategy_knowledge", map[string]any{
		"strategy_name": name,
		"content":       content,
		"folder_path":   tools.StrategyArchiveFolder,
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
	for time.Now().Before(deadline) {
		if hits := r.searchStrategyArchiveHits(ctx, query, toolCtx, recordTool); len(hits) > 0 {
			flow.VerifySnippet = cognitionVerifySnippet(hits, flow)
			flow.Phase = PhaseSummarize
			flow.touch()
			return nil
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		time.Sleep(2 * time.Second)
	}
	return terminalError("知识库读回验证超时：search_knowledge 未命中，请稍后重试")
}

func cognitionVerifySnippet(hits []any, flow *Flow) string {
	title := strings.TrimSpace(flow.KnowledgeTitle)
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content := hitFieldString(row, "content")
		if content == "" || !isStrategyArchiveContent(content) {
			continue
		}
		if title != "" {
			hitTitle := hitFieldString(row, "title", "filename")
			if hitTitle != "" && hitTitle != title && !strings.Contains(hitTitle, strategyDisplayLabel(flow)) {
				continue
			}
		}
		return truncate(content, 240)
	}
	if preview := cognitionDraftPreview(flow.KBDraft); preview != "" {
		return preview
	}
	return firstHitPreview(hits)
}

func cognitionDraftPreview(draft string) string {
	draft = strings.TrimSpace(draft)
	if draft == "" {
		return ""
	}
	if idx := strings.Index(draft, "## 一句话定位"); idx >= 0 {
		return truncate(strings.TrimSpace(draft[idx:]), 240)
	}
	return truncate(draft, 240)
}

func hitFieldString(row map[string]any, keys ...string) string {
	for _, key := range keys {
		if s := catalogStringValue(row[key]); s != "" {
			return s
		}
	}
	return ""
}

func firstHitPreview(hits []any) string {
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if content := hitFieldString(row, "content"); content != "" {
			return truncate(content, 240)
		}
	}
	return ""
}

func renderGenerateCognitionReport(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	var b strings.Builder
	fmt.Fprintf(&b, "## 策略档案已生成 · %s\n\n", label)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 策略库 | %s（%s） |\n", label, flow.CatalogType)
	fmt.Fprintf(&b, "| 档案合成 | LLM 基于策略库 |\n")
	if flow.KnowledgeID != "" {
		fmt.Fprintf(&b, "| 知识库 | [%s](kb:%s) · %s |\n", flow.KnowledgeTitle, flow.KnowledgeID, tools.StrategyArchiveFolder)
	}
	if flow.VerifySnippet != "" {
		fmt.Fprintf(&b, "\n**读回验证片段**：\n\n> %s\n", flow.VerifySnippet)
	}
	fmt.Fprintf(&b, "\n> 策略档案已写入知识库；策略开发 workflow 将从此处读取。")
	return strings.TrimSpace(b.String())
}

func renderGenerateCognitionPartial(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	return fmt.Sprintf("## 生成策略档案（进行中）\n\n- 策略：%s\n- 阶段：%s",
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
