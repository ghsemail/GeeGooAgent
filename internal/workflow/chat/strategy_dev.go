package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var readStrategyDevMessagePattern = regexp.MustCompile(`(?i)^读取\s*(.+?)\s*策略$`)

// FormatReadStrategyDevMessage is the canonical Dock utterance for strategy_dev workflow.
func FormatReadStrategyDevMessage(strategyName string) string {
	name := strings.TrimSpace(strategyName)
	if name == "" {
		return "读取策略"
	}
	return fmt.Sprintf("读取 %s 策略", name)
}

func newStrategyDevFlow(userText string) *Flow {
	now := time.Now().UTC()
	query := extractStrategyQuery(userText)
	return &Flow{
		RunID:         newRunID(),
		Template:      SkillStrategyDev,
		Status:        StatusRunning,
		WorkflowStep:  StepStrategyDevelop,
		Phase:         PhaseDevPick,
		StrategyQuery: query,
		TriggerText:   strings.TrimSpace(userText),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (r *Runner) advanceStrategyDev(
	ctx context.Context,
	_ *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	switch flow.Phase {
	case PhaseDevPick:
		if strings.TrimSpace(flow.StrategyQuery) == "" {
			return terminalError("请说明要读取的策略名称，例如：读取 Macd4H 策略")
		}
		flow.Phase = PhaseDevEnsureArchive
		flow.touch()
		return nil
	case PhaseDevEnsureArchive, PhaseDevReadCognition:
		return r.phaseDevEnsureArchive(ctx, flow, toolCtx, recordTool)
	case PhaseSummarize, PhaseDone:
		return nil
	default:
		flow.Phase = PhaseDevPick
		return nil
	}
}

// phaseDevEnsureArchive loads strategy archive from KB; generates one if missing.
func (r *Runner) phaseDevEnsureArchive(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	query := strings.TrimSpace(flow.StrategyQuery)
	if hits := r.searchStrategyArchiveHits(ctx, query, toolCtx, recordTool); len(hits) > 0 {
		r.loadArchiveFromHits(flow, hits, query)
		flow.DevArchiveGenerated = false
		flow.Phase = PhaseSummarize
		flow.touch()
		return nil
	}
	if err := r.ensureStrategyArchiveGenerated(ctx, flow, toolCtx, recordTool); err != nil {
		return err
	}
	flow.DevArchiveGenerated = true
	if flow.CatalogLabel == "" {
		flow.CatalogLabel = query
	}
	flow.Phase = PhaseSummarize
	flow.touch()
	return nil
}

func (r *Runner) loadArchiveFromHits(flow *Flow, hits []any, query string) {
	flow.KBDraft = joinHitContents(hits, 6000)
	flow.VerifySnippet = firstHitPreview(hits)
	if title := hitField(hits, "title"); title != "" {
		flow.KnowledgeTitle = title
	} else if title := hitField(hits, "filename"); title != "" {
		flow.KnowledgeTitle = title
	}
	flow.CatalogLabel = query
}

// ensureStrategyArchiveGenerated runs generate_strategy_cognition pipeline inline.
func (r *Runner) ensureStrategyArchiveGenerated(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	query := strings.TrimSpace(flow.StrategyQuery)
	match, err := resolveStrategyCatalog(ctx, query, toolCtx, r.runToolCall(recordTool))
	if err != nil {
		return fmt.Errorf("策略库未找到「%s」，无法自动生成策略档案：%w", query, err)
	}
	flow.CatalogType = match.Type
	flow.CatalogLabel = match.Label
	flow.CatalogRaw = match.Raw
	if query == "" {
		flow.StrategyQuery = match.Label
	}
	if err := r.phaseCognitionCompose(ctx, flow, toolCtx, recordTool); err != nil {
		return err
	}
	if err := r.phaseCognitionSaveKB(ctx, flow, toolCtx, recordTool); err != nil {
		return err
	}
	if err := r.phaseCognitionVerifyKB(ctx, flow, toolCtx, recordTool); err != nil {
		return err
	}
	if strings.TrimSpace(flow.KBDraft) == "" {
		return terminalError("策略档案生成完成但缺少正文")
	}
	return nil
}

func joinHitContents(hits []any, maxRunes int) string {
	var b strings.Builder
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		content := strings.TrimSpace(fmt.Sprint(row["content"]))
		if content == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n---\n\n")
		}
		b.WriteString(content)
		if len([]rune(b.String())) >= maxRunes {
			break
		}
	}
	return truncate(b.String(), maxRunes)
}

func hitField(hits []any, key string) string {
	for _, raw := range hits {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if v := strings.TrimSpace(fmt.Sprint(row[key])); v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func renderStrategyDevReport(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	var b strings.Builder
	fmt.Fprintf(&b, "## %s · 策略开发\n\n", FormatReadStrategyDevMessage(label))
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	if flow.DevArchiveGenerated {
		fmt.Fprintf(&b, "| 策略档案 | 知识库无记录 → 已自动生成并写入 %s |\n", tools.StrategyArchiveFolder)
	} else {
		fmt.Fprintf(&b, "| 策略档案 | 知识库已有记录 → 直接读取 %s |\n", tools.StrategyArchiveFolder)
	}
	if flow.KnowledgeTitle != "" {
		fmt.Fprintf(&b, "| 文档 | %s |\n", flow.KnowledgeTitle)
	}
	fmt.Fprintf(&b, "| 载入方式 | search_knowledge / 策略档案 workflow |\n")
	if flow.VerifySnippet != "" {
		fmt.Fprintf(&b, "\n**档案摘要**：\n\n> %s\n", flow.VerifySnippet)
	}
	fmt.Fprintf(&b, "\n> 策略档案已注入开发上下文；后续 Step（回测/调参/实现）将在此 workflow 扩展。")
	return strings.TrimSpace(b.String())
}

func renderStrategyDevPartial(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	phaseLabel := flow.Phase
	switch flow.Phase {
	case PhaseDevEnsureArchive, PhaseDevReadCognition:
		phaseLabel = "检查/加载策略档案"
	}
	return fmt.Sprintf("## %s\n\n- 阶段：%s",
		FormatReadStrategyDevMessage(label), phaseLabel)
}
