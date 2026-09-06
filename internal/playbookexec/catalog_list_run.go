package playbookexec

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func (r *Router) runSignalCatalog(ctx context.Context, in Input) runtime.TurnResult {
	records := []runtime.StepRecord{}
	step := in.StepBase
	if step <= 0 {
		step = 1
	}
	emit := in.OnProgress
	recordPlan := func(summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "plan",
			Summary: strings.TrimSpace(summary),
		})
		if emit != nil {
			emit("playbook_exec", map[string]any{
				"playbook": "signal-catalog-list", "phase": "plan", "summary": summary,
			})
		}
	}
	recordTool := func(name, status, summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: name, ToolStatus: status, Summary: summary,
		})
	}

	recordPlan("列举可用组合信号与单指标信号（catalog SOP）")

	toolCtx := in.ToolCtx
	toolCtx.FullCatalogPayload = true
	toolCtx.Step = step
	if emit != nil {
		toolCtx.Progress = func(event string, data map[string]any) {
			emit(event, data)
		}
	}

	comboRes := r.runTool(ctx, toolCtx, "get_signal_combinations", map[string]any{}, recordTool)
	if comboRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("拉取组合信号失败：%s", comboRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: comboRes.Summary, StepRecords: records}
	}
	indexRes := r.runTool(ctx, toolCtx, "get_index_signals", map[string]any{}, recordTool)
	if indexRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("拉取单指标信号失败：%s", indexRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: indexRes.Summary, StepRecords: records}
	}

	reply := formatSignalCatalogReply(catalogItemsFromResult(comboRes), catalogItemsFromResult(indexRes))
	in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
	})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}
}

func catalogItemsFromResult(res tools.Result) []map[string]any {
	return slots.CatalogItems(res.Data)
}

func formatSignalCatalogReply(combos, indices []map[string]any) string {
	if len(combos) == 0 && len(indices) == 0 {
		return "暂无可用信号策略，请稍后再试。"
	}
	var b strings.Builder
	b.WriteString("## 可用信号策略\n\n")
	if len(combos) > 0 {
		b.WriteString(fmt.Sprintf("### 组合信号（%d 个）\n\n", len(combos)))
		if freq := catalogFrequencyHint(combos); freq != "" {
			b.WriteString(freq)
			b.WriteString("\n\n")
		}
		for _, row := range combos {
			b.WriteString(formatCatalogBullet(row))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	if len(indices) > 0 {
		b.WriteString(fmt.Sprintf("### 单指标信号（%d 个）\n\n", len(indices)))
		for _, row := range indices {
			b.WriteString(formatCatalogBullet(row))
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	b.WriteString("如需测买卖点，请指定组合名（如「SAR+MACD」）和标的；如需回测请说明「回测」。")
	return strings.TrimSpace(b.String())
}

func formatCatalogBullet(row map[string]any) string {
	name := strings.TrimSpace(fmt.Sprint(row["name"]))
	if name == "" {
		name = strings.TrimSpace(fmt.Sprint(row["index"]))
	}
	brief := strings.TrimSpace(fmt.Sprint(row["brief"]))
	if brief == "" {
		brief = strings.TrimSpace(fmt.Sprint(row["info"]))
	}
	if brief != "" {
		return fmt.Sprintf("- **%s**：%s", name, brief)
	}
	return fmt.Sprintf("- **%s**", name)
}

func catalogFrequencyHint(rows []map[string]any) string {
	seen := map[string]struct{}{}
	for _, row := range rows {
		f := strings.TrimSpace(fmt.Sprint(row["frequency"]))
		if f == "" {
			continue
		}
		for _, part := range strings.FieldsFunc(f, func(r rune) bool {
			return r == '/' || r == '、' || r == ','
		}) {
			part = strings.TrimSpace(part)
			if part != "" {
				seen[part] = struct{}{}
			}
		}
	}
	if len(seen) == 0 {
		return "组合信号通常支持 5m / 60m / daily 等频率。"
	}
	parts := make([]string, 0, len(seen))
	for f := range seen {
		parts = append(parts, f)
	}
	sort.Strings(parts)
	return fmt.Sprintf("支持频率：%s", strings.Join(parts, " / "))
}
