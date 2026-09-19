package chat

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var signalDiagnoseMessagePattern = regexp.MustCompile(`(?i)^(?:信号诊断|诊断)\s*(.+?)(?:\s*[·\.]\s*(.+))?$`)

// FormatSignalDiagnoseMessage is the canonical Dock utterance for signal_diagnose workflow.
func FormatSignalDiagnoseMessage(strategyName, stockName string) string {
	strategy := strings.TrimSpace(strategyName)
	stock := strings.TrimSpace(stockName)
	switch {
	case strategy != "" && stock != "":
		return fmt.Sprintf("诊断 %s 策略 · %s", strategy, stock)
	case strategy != "":
		return fmt.Sprintf("诊断 %s 策略", strategy)
	default:
		return "信号诊断"
	}
}

func newSignalDiagnoseFlow(userText string) *Flow {
	now := time.Now().UTC()
	strategy, stock := parseSignalDiagnoseMessage(userText)
	return &Flow{
		RunID:         newRunID(),
		Template:      SkillSignalDiagnose,
		Status:        StatusRunning,
		Phase:         PhaseSignalDiagPick,
		StrategyQuery: strategy,
		StockQuery:    stock,
		MonthsBack:    defaultMonthsBack,
		TriggerText:   strings.TrimSpace(userText),
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func parseSignalDiagnoseMessage(text string) (strategy, stock string) {
	text = strings.TrimSpace(text)
	if m := signalDiagnoseMessagePattern.FindStringSubmatch(text); len(m) > 1 {
		strategy = strings.TrimSpace(m[1])
		if len(m) > 2 {
			stock = strings.TrimSpace(m[2])
		}
	}
	if strategy == "" {
		strategy = extractSignalDiagnoseStrategy(text)
	}
	if stock == "" {
		stock = slots.ExtractExplicitStockReference(text)
	}
	strategy = strings.TrimSpace(strings.TrimSuffix(strategy, "策略"))
	return strings.TrimSpace(strategy), strings.TrimSpace(stock)
}

func extractSignalDiagnoseStrategy(text string) string {
	repl := strings.NewReplacer(
		"信号诊断", " ", "诊断", " ", "策略", " ",
		"帮我", " ", "请", " ", "workflow", " ", "Workflow", " ",
	)
	clean := strings.TrimSpace(repl.Replace(text))
	if clean == "" {
		return strings.TrimSpace(text)
	}
	for _, tok := range strings.Fields(clean) {
		if isLikelyStockToken(tok) {
			continue
		}
		if len([]rune(tok)) >= 2 {
			return tok
		}
	}
	return clean
}

func isLikelyStockToken(tok string) bool {
	upper := strings.ToUpper(tok)
	if strings.Contains(tok, "腾讯") || strings.Contains(tok, "小米") || strings.Contains(tok, "美团") {
		return true
	}
	return strings.Contains(upper, ".HK") || strings.Contains(upper, ".US") || strings.Contains(upper, ".SH")
}

func (r *Runner) advanceSignalDiagnose(
	ctx context.Context,
	session *runtime.Session,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	switch flow.Phase {
	case PhaseSignalDiagPick:
		if strings.TrimSpace(flow.StrategyQuery) == "" {
			return terminalError("请说明要诊断的策略，例如：诊断 SAR 策略 · 腾讯")
		}
		if strings.TrimSpace(flow.StockQuery) == "" {
			return terminalError("请说明标的，例如：诊断 SAR 策略 · 腾讯")
		}
		flow.Phase = PhaseReadStrategy
		flow.touch()
		return nil
	case PhaseReadStrategy:
		return r.phaseSignalDiagnoseReadStrategy(ctx, flow, toolCtx, recordTool)
	case PhaseResolveSymbol:
		if err := r.phaseResolveSymbol(ctx, session, flow, toolCtx, recordTool); err != nil {
			return err
		}
		if flow.Phase == PhasePickStrategies {
			flow.Phase = PhaseRunProbe
			flow.touch()
		}
		return nil
	case PhaseRunProbe:
		return r.phaseSignalDiagnoseRunProbe(ctx, flow, toolCtx, recordTool)
	case PhaseBuildDetail:
		return r.phaseSignalDiagnoseBuildDetail(ctx, flow, toolCtx, recordTool)
	case PhaseSummarize, PhaseDone:
		return nil
	default:
		flow.Phase = PhaseSignalDiagPick
		return nil
	}
}

func (r *Runner) phaseSignalDiagnoseReadStrategy(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	runTool := r.runToolCall(recordTool)
	match, err := matchIndex(ctx, flow.StrategyQuery, toolCtx, runTool)
	if err != nil {
		match, err = resolveStrategyCatalog(ctx, flow.StrategyQuery, toolCtx, runTool)
	}
	if err != nil {
		return err
	}
	flow.CatalogType = match.Type
	flow.CatalogLabel = match.Label
	flow.CatalogRaw = match.Raw
	flow.Phase = PhaseResolveSymbol
	flow.touch()
	return nil
}

func (r *Runner) phaseSignalDiagnoseRunProbe(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	sig, err := r.resolveDiagnoseSignal(ctx, flow, toolCtx, recordTool)
	if err != nil {
		return err
	}
	months := flow.MonthsBack
	if months <= 0 {
		months = defaultMonthsBack
	}
	frequency := pickProbeFrequency(sig.Frequency, flow.CatalogRaw)
	res := r.runTool(ctx, toolCtx, "probe_bot_signal_series", map[string]any{
		"code":        flow.StockCode,
		"frequency":   frequency,
		"buy_signal":  sig.Buy,
		"sell_signal": sig.Sell,
		"months_back": months,
	}, recordTool)
	if res.Status != tools.StatusOK {
		return fmt.Errorf("probe 失败：%s", res.Summary)
	}
	flow.ProbeBuyHits = probeHitCount(res.Data, "buy_hits", "buy_merged", 1)
	flow.ProbeSellHits = probeHitCount(res.Data, "sell_hits", "sell_merged", -1)
	flow.ProbeBarCount = intAny(res.Data["bar_count"])
	if flow.ProbeBarCount == 0 {
		if bars, ok := res.Data["bars"].([]any); ok {
			flow.ProbeBarCount = len(bars)
		}
	}
	if flow.Strategies == nil {
		flow.Strategies = []StrategyItem{{}}
	}
	item := &flow.Strategies[0]
	item.Label = sig.StrategyLabel
	item.Query = flow.StrategyQuery
	item.Frequency = frequency
	item.MonthsBack = months
	item.BuyHits = flow.ProbeBuyHits
	item.SellHits = flow.ProbeSellHits
	item.Summary = strings.TrimSpace(res.Summary)
	item.Status = StepDone
	flow.Phase = PhaseBuildDetail
	flow.touch()
	return nil
}

func (r *Runner) phaseSignalDiagnoseBuildDetail(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) error {
	sig, err := r.resolveDiagnoseSignal(ctx, flow, toolCtx, recordTool)
	if err != nil {
		return err
	}
	months := flow.MonthsBack
	if months <= 0 {
		months = defaultMonthsBack
	}
	frequency := sig.Frequency
	if len(flow.Strategies) > 0 && strings.TrimSpace(flow.Strategies[0].Frequency) != "" {
		frequency = flow.Strategies[0].Frequency
	}
	frequency = pickProbeFrequency(frequency, flow.CatalogRaw)
	res := r.runTool(ctx, toolCtx, "diagnose_bot_signal_series", map[string]any{
		"code":        flow.StockCode,
		"frequency":   frequency,
		"buy_signal":  sig.Buy,
		"sell_signal": sig.Sell,
		"months_back": months,
		"side":        "both",
	}, recordTool)
	if res.Status != tools.StatusOK {
		return fmt.Errorf("diagnose 失败：%s", res.Summary)
	}
	flow.DiagnoseRaw = res.Data
	flow.DiagnoseVerdict = strings.TrimSpace(fmt.Sprint(res.Data["verdict"]))
	flow.DiagnoseSummary = strings.TrimSpace(fmt.Sprint(res.Data["summary"]))
	if hits, ok := res.Data["hits"].(map[string]any); ok {
		if flow.ProbeBuyHits == 0 {
			flow.ProbeBuyHits = intAny(hits["buy"])
		}
		if flow.ProbeSellHits == 0 {
			flow.ProbeSellHits = intAny(hits["sell"])
		}
	}
	flow.Phase = PhaseSummarize
	flow.touch()
	return nil
}

func (r *Runner) resolveDiagnoseSignal(
	ctx context.Context,
	flow *Flow,
	toolCtx tools.Context,
	recordTool func(name, status, summary string),
) (slots.ResolvedSignal, error) {
	kind := "combination"
	switch flow.CatalogType {
	case catalogTypeIndex, catalogTypeDefinition:
		kind = "indicator"
	}
	plan := slots.SignalPlan{
		SignalQuery: coalesceStrings(flow.CatalogLabel, flow.StrategyQuery),
		SignalKind:  kind,
	}
	return slots.ResolveSignal(ctx, toolCtx, r.runToolCall(recordTool), plan)
}

func probeHitCount(data map[string]any, compactKey, mergedKey string, target int) int {
	if _, ok := data[compactKey]; ok {
		return intAny(data[compactKey])
	}
	if merged, ok := data[mergedKey].([]any); ok {
		n := 0
		for _, v := range merged {
			switch t := v.(type) {
			case int:
				if t == target {
					n++
				}
			case float64:
				if int(t) == target {
					n++
				}
			}
		}
		return n
	}
	return 0
}

func renderSignalDiagnosePartial(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	title := FormatSignalDiagnoseMessage(label, coalesceStrings(flow.StockName, flow.StockQuery))
	var b strings.Builder
	fmt.Fprintf(&b, "## %s\n\n", title)
	fmt.Fprintf(&b, "- 阶段：%s\n", signalDiagnosePhaseLabel(flow.Phase))
	if flow.StockCode != "" {
		fmt.Fprintf(&b, "- 标的：%s %s\n", flow.StockName, flow.StockCode)
	}
	if flow.ProbeBuyHits > 0 || flow.ProbeSellHits > 0 || flow.Phase == PhaseBuildDetail || flow.Phase == PhaseSummarize {
		fmt.Fprintf(&b, "- 信号测试：买 %d / 卖 %d\n", flow.ProbeBuyHits, flow.ProbeSellHits)
	}
	return strings.TrimSpace(b.String())
}

func renderSignalDiagnoseReport(flow *Flow) string {
	label := strategyDisplayLabel(flow)
	title := FormatSignalDiagnoseMessage(label, coalesceStrings(flow.StockName, flow.StockQuery))
	var b strings.Builder
	fmt.Fprintf(&b, "## %s · 信号诊断\n\n", title)
	fmt.Fprintf(&b, "| 步骤 | 结果 |\n| --- | --- |\n")
	fmt.Fprintf(&b, "| 读取策略 | %s（%s） |\n", label, flow.CatalogType)
	fmt.Fprintf(&b, "| 标的 | %s %s |\n", flow.StockName, flow.StockCode)
	freq := "-"
	if len(flow.Strategies) > 0 && flow.Strategies[0].Frequency != "" {
		freq = flow.Strategies[0].Frequency
	}
	fmt.Fprintf(&b, "| 周期 / 回溯 | %s / %d 月 |\n", freq, flow.MonthsBack)
	fmt.Fprintf(&b, "| 信号测试 probe | 买 %d 次 / 卖 %d 次", flow.ProbeBuyHits, flow.ProbeSellHits)
	if flow.ProbeBarCount > 0 {
		fmt.Fprintf(&b, "（%d 根 K 线）", flow.ProbeBarCount)
	}
	fmt.Fprintf(&b, " |\n")
	if flow.DiagnoseVerdict != "" {
		fmt.Fprintf(&b, "| 诊断结论 | %s |\n", flow.DiagnoseVerdict)
	}
	if flow.DiagnoseSummary != "" {
		fmt.Fprintf(&b, "\n**摘要**：%s\n", flow.DiagnoseSummary)
	}
	appendDiagnoseRuleDetails(&b, flow.DiagnoseRaw)
	fmt.Fprintf(&b, "\n> 由 workflow 执行：read_strategy → probe_bot_signal_series → diagnose_bot_signal_series。")
	return strings.TrimSpace(b.String())
}

func appendDiagnoseRuleDetails(b *strings.Builder, raw map[string]any) {
	if len(raw) == 0 {
		return
	}
	for _, side := range []struct {
		key, title string
	}{
		{"buy_rules", "买入规则"},
		{"sell_rules", "卖出规则"},
	} {
		rules, ok := raw[side.key].([]any)
		if !ok || len(rules) == 0 {
			continue
		}
		fmt.Fprintf(b, "\n### %s\n\n", side.title)
		for i, item := range rules {
			row, ok := item.(map[string]any)
			if !ok {
				continue
			}
			name := fmt.Sprint(row["index"])
			fmt.Fprintf(b, "%d. **%s**（%s）\n", i+1, name, row["type"])
			if param, ok := row["param"].(map[string]any); ok && len(param) > 0 {
				fmt.Fprintf(b, "   - 参数：%s\n", formatProbeParam(param))
			}
			if algo := strings.TrimSpace(fmt.Sprint(row["algorithm"])); algo != "" && algo != "<nil>" {
				fmt.Fprintf(b, "   - 算法：%s\n", algo)
			}
			if n := intAny(row["trigger_count"]); n >= 0 {
				fmt.Fprintf(b, "   - 规则触发：%d 次\n", n)
			}
			if stats, ok := row["stats"].(map[string]any); ok {
				if cols, ok := stats["columns"].([]any); ok && len(cols) > 0 {
					if col, ok := cols[0].(map[string]any); ok {
						fmt.Fprintf(b, "   - 指标区间：%v min=%v max=%v last=%v\n",
							col["name"], col["min"], col["max"], col["last"])
					}
				}
			}
			if lb, ok := row["last_bar"].(map[string]any); ok {
				fmt.Fprintf(b, "   - 最后一根：signal=%v reason=%v\n", lb["signal"], lb["reason"])
			}
			if samples, ok := row["sample_reasons"].([]any); ok && len(samples) > 0 {
				fmt.Fprintf(b, "   - 样例触发：\n")
				for _, s := range samples {
					if m, ok := s.(map[string]any); ok {
						fmt.Fprintf(b, "     - %v · %v\n", m["time"], m["reason"])
					}
				}
			}
		}
	}
}

func formatProbeParam(param map[string]any) string {
	parts := make([]string, 0, len(param))
	for k, v := range param {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

func signalDiagnosePhaseLabel(phase string) string {
	switch phase {
	case PhaseSignalDiagPick:
		return "确认策略与标的"
	case PhaseReadStrategy:
		return "读取策略库"
	case PhaseResolveSymbol:
		return "解析标的"
	case PhaseRunProbe:
		return "信号测试 probe"
	case PhaseBuildDetail:
		return "读取诊断明细"
	case PhaseSummarize:
		return "汇总"
	default:
		return phase
	}
}

func pickProbeFrequency(raw string, catalogRaw map[string]any) string {
	raw = strings.TrimSpace(raw)
	if catalogRaw != nil {
		if idx := strings.TrimSpace(fmt.Sprint(catalogRaw["index"])); strings.EqualFold(idx, "Macd4HRhythm") {
			return "60m"
		}
		if strings.Contains(strings.ToUpper(fmt.Sprint(catalogRaw["name"])), "RESONANCE") {
			return "15m"
		}
	}
	for _, pref := range []string{"60m", "15m", "5m", "daily"} {
		if strings.Contains(raw, pref) {
			return pref
		}
	}
	if raw == "" || strings.HasPrefix(raw, "[") || strings.Contains(raw, " ") {
		return "60m"
	}
	return raw
}

func coalesceStrings(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
