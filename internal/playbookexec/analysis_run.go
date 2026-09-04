package playbookexec

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// AnalysisRunPlan is the structured plan for stock analysis SOP.
type AnalysisRunPlan struct {
	StockQuery string
	Period     string // daily | weekly
	Tag        string // price | kline | flag | capital_flow | index
	IndexName  string // MACD, RSI, etc. when Tag=index
}

func (r *Router) runAnalysis(ctx context.Context, in Input) runtime.TurnResult {
	records := []runtime.StepRecord{}
	step := in.StepBase
	if step <= 0 {
		step = 1
	}
	emit := in.OnProgress
	recordTool := func(name, status, summary string) {
		records = append(records, runtime.StepRecord{
			Step: step, Timestamp: time.Now().UTC(), Kind: "tool",
			ToolName: name, ToolStatus: status, Summary: summary,
		})
	}

	plan := heuristicAnalysisPlan(in.UserText)
	if strings.TrimSpace(plan.StockQuery) == "" {
		msg := "请说明要分析哪只股票或标的"
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}

	toolCtx := in.ToolCtx
	toolCtx.FullCatalogPayload = true
	toolCtx.Step = step
	if emit != nil {
		toolCtx.Progress = func(event string, data map[string]any) {
			emit(event, data)
		}
	}

	runTool := func(ctx context.Context, req tools.CallRequest, tc tools.Context) tools.Result {
		return r.runTool(ctx, tc, req.Name, req.Arguments, recordTool)
	}
	code, name, _, err := slots.ResolveStock(ctx, toolCtx, runTool, plan.StockQuery)
	if err != nil {
		msg := err.Error()
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}

	promptID, templateName, err := r.resolveAnalysisTemplate(ctx, toolCtx, plan, recordTool)
	if err != nil {
		msg := err.Error()
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: msg, StepRecords: records}
	}

	analysisRes := r.runTool(ctx, toolCtx, "get_mcp_analysis", map[string]any{
		"name":      name,
		"code":      code,
		"prompt_id": promptID,
		"period":    plan.Period,
		"language":  "cn",
	}, recordTool)
	if analysisRes.Status != tools.StatusOK {
		msg := fmt.Sprintf("分析执行失败：%s", analysisRes.Summary)
		in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: msg})
		return runtime.TurnResult{AssistantText: msg, Failed: true, Error: analysisRes.Summary, StepRecords: records}
	}

	reply := formatAnalysisReply(name, code, templateName, plan.Period, analysisRes)
	in.Session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: reply})
	records = append(records, runtime.StepRecord{
		Step: step, Timestamp: time.Now().UTC(), Kind: "reply", Summary: truncate(reply, 300),
	})
	return runtime.TurnResult{AssistantText: reply, StepRecords: records}
}

func heuristicAnalysisPlan(message string) AnalysisRunPlan {
	plan := AnalysisRunPlan{Period: "daily", Tag: "price"}
	msg := strings.TrimSpace(message)
	plan.StockQuery = slots.ExtractStockQuery(msg)
	lower := strings.ToLower(msg)
	if analysisHasAny(msg, []string{"这周", "周线", "weekly"}) {
		plan.Period = "weekly"
	}
	switch {
	case analysisHasAny(msg, []string{"资金", "主力", "资金流", "capital"}):
		plan.Tag = "capital_flow"
	case analysisHasAny(msg, []string{"K线", "k线", "形态", "蜡烛"}):
		plan.Tag = "kline"
	case analysisHasAny(msg, []string{"趋势", "走势", "flag"}):
		plan.Tag = "flag"
	case strings.Contains(lower, "macd"):
		plan.Tag, plan.IndexName = "index", "MACD"
	case strings.Contains(lower, "rsi"):
		plan.Tag, plan.IndexName = "index", "RSI"
	case strings.Contains(lower, "sar"):
		plan.Tag, plan.IndexName = "index", "SAR"
	case strings.Contains(lower, "ema"):
		plan.Tag, plan.IndexName = "index", "EMA"
	case strings.Contains(lower, "kdj"):
		plan.Tag, plan.IndexName = "index", "KDJ"
	case strings.Contains(lower, "boll"):
		plan.Tag, plan.IndexName = "index", "BOLL"
	}
	return plan
}

func (r *Router) resolveAnalysisTemplate(
	ctx context.Context,
	toolCtx tools.Context,
	plan AnalysisRunPlan,
	recordTool func(name, status, summary string),
) (promptID, templateName string, err error) {
	if plan.Tag == "index" && strings.TrimSpace(plan.IndexName) != "" {
		res := r.runTool(ctx, toolCtx, "get_single_prompt_template_by_index", map[string]any{
			"index":  plan.IndexName,
			"period": plan.Period,
		}, recordTool)
		if res.Status != tools.StatusOK {
			return "", "", fmt.Errorf("指标模板查询失败：%s", res.Summary)
		}
		return pickPromptFromData(res.Data)
	}
	res := r.runTool(ctx, toolCtx, "get_single_prompt_template", map[string]any{
		"type": "tech",
		"tag":  plan.Tag,
	}, recordTool)
	if res.Status != tools.StatusOK {
		return "", "", fmt.Errorf("技术面模板查询失败：%s", res.Summary)
	}
	return pickPromptFromData(res.Data)
}

func pickPromptFromData(data map[string]any) (promptID, templateName string, err error) {
	if id := strings.TrimSpace(fmt.Sprint(data["selected_prompt_id"])); id != "" && id != "<nil>" {
		name := strings.TrimSpace(fmt.Sprint(data["selected_prompt_name"]))
		if name == "" || name == "<nil>" {
			name = strings.TrimSpace(fmt.Sprint(data["name"]))
		}
		return id, name, nil
	}
	items := catalogItems(data)
	if len(items) == 0 {
		return "", "", fmt.Errorf("未找到可用分析模板")
	}
	row := items[0]
	id := strings.TrimSpace(fmt.Sprint(row["prompt_id"]))
	if id == "" {
		id = strings.TrimSpace(fmt.Sprint(row["_id"]))
	}
	name := strings.TrimSpace(fmt.Sprint(row["name"]))
	if id == "" {
		return "", "", fmt.Errorf("模板缺少 prompt_id")
	}
	return id, name, nil
}

func formatAnalysisReply(name, code, templateName, period string, res tools.Result) string {
	body := strings.TrimSpace(fmt.Sprint(res.Data["analysis_result"]))
	if body == "" {
		body = strings.TrimSpace(res.Summary)
	}
	if body == "" {
		body = "分析完成，但未返回正文。"
	}
	label := templateName
	if label == "" {
		label = "技术面"
	}
	return fmt.Sprintf("## %s %s · %s（%s）\n\n%s\n\n> 由 playbook 确定性执行（get_mcp_analysis）",
		name, code, label, period, body)
}

func analysisHasAny(msg string, tokens []string) bool {
	lower := strings.ToLower(msg)
	for _, tok := range tokens {
		if tok == "" {
			continue
		}
		if strings.Contains(msg, tok) || strings.Contains(lower, strings.ToLower(tok)) {
			return true
		}
	}
	return false
}
