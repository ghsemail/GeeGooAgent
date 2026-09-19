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
			return terminalError("请说明要开发的策略名称，例如：策略开发 Macd4H")
		}
		flow.Phase = PhaseDevReadCognition
		flow.touch()
		return nil
	case PhaseDevReadCognition:
		return r.phaseDevReadCognition(ctx, flow, toolCtx, recordTool)
	case PhaseSummarize, PhaseDone:
		return nil
	default:
		flow.Phase = PhaseDevPick
		return nil
	}
}

func (r *Runner) phaseDevReadCognition(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	query := strings.TrimSpace(flow.StrategyQuery)
	hits := r.searchStrategyArchiveHits(ctx, query, toolCtx, recordTool)
	if len(hits) == 0 {
		return terminalError(fmt.Sprintf(
			"知识库中尚无「%s」的策略档案，请先发送：%s",
			query, FormatGenerateStrategyArchiveMessage(query),
		))
	}
	flow.KBDraft = joinHitContents(hits, 6000)
	flow.VerifySnippet = firstHitPreview(hits)
	if title := hitField(hits, "title"); title != "" {
		flow.KnowledgeTitle = title
	} else if title := hitField(hits, "filename"); title != "" {
		flow.KnowledgeTitle = title
	}
	flow.CatalogLabel = query
	flow.Phase = PhaseSummarize
	flow.touch()
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
	fmt.Fprintf(&b, "## 策略开发 · 已加载策略档案 · %s\n\n", label)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 知识库 | %s · %s |\n", flow.KnowledgeTitle, tools.StrategyArchiveFolder)
	fmt.Fprintf(&b, "| 载入方式 | search_knowledge 读取策略档案 |\n")
	if flow.VerifySnippet != "" {
		fmt.Fprintf(&b, "\n**档案摘要**：\n\n> %s\n", flow.VerifySnippet)
	}
	fmt.Fprintf(&b, "\n> 策略档案已从知识库载入；后续 Step（回测/调参/实现）将在此 workflow 扩展。")
	return strings.TrimSpace(b.String())
}

func renderStrategyDevPartial(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	return fmt.Sprintf("## 策略开发（进行中）\n\n- 策略：%s\n- 阶段：%s",
		label, flow.Phase)
}
