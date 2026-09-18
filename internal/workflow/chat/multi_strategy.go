package chat

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/sessiontask"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func newMultiStrategyFlow(userText string, session *runtime.Session) *Flow {
	now := time.Now().UTC()
	st := sessiontask.BuildState(session, "signal_probe")
	stockQuery := slots.ExtractExplicitStockReference(userText)
	if stockQuery == "" && st.Symbol != "" {
		stockQuery = st.Symbol
	}
	if stockQuery == "" {
		stockQuery = stockQueryFromSession(session)
	}
	flow := &Flow{
		RunID:       newRunID(),
		Template:    SkillMultiStrategyCompare,
		Status:      StatusRunning,
		Phase:       PhaseResolveSymbol,
		MonthsBack:  defaultMonthsBack,
		StockQuery:  stockQuery,
		TriggerText: strings.TrimSpace(userText),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if !IsContinueAllIntent(userText) || countStrategyMentions(userText) >= 2 {
		flow.Strategies = extractStrategyQueries(userText, session)
	}
	return flow
}

func stockQueryFromSession(session *runtime.Session) string {
	if session == nil {
		return ""
	}
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role != llm.RoleUser {
			continue
		}
		if q := slots.ExtractExplicitStockReference(session.Messages[i].Content); q != "" {
			return q
		}
	}
	return ""
}

func extractStrategyQueries(userText string, session *runtime.Session) []StrategyItem {
	queries := splitStrategyQueries(userText)
	if len(queries) == 0 {
		return nil
	}
	out := make([]StrategyItem, 0, len(queries))
	for _, q := range queries {
		item := StrategyItem{Query: q, Status: StepPending, MonthsBack: defaultMonthsBack}
		sp := slots.SignalPlan{SignalQuery: q}
		slots.ApplySignalHeuristics(&sp, userText)
		item.Kind = sp.SignalKind
		if strings.TrimSpace(item.Kind) == "" {
			item.Kind = "combination"
		}
		applyStrategyFrequency(&item)
		out = append(out, item)
	}
	return out
}

func splitStrategyQueries(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	repl := strings.NewReplacer("对比", " ", "哪个", " ", "信号多", " ", "买卖点", " ", "测一下", " ", "跑一下", " ")
	clean := repl.Replace(text)
	repl = strings.NewReplacer("和", " ", "与", " ", " VS ", " ", " vs ", " ", "、", " ", "，", " ", ",", " ", "/", " ", "|", " ")
	clean = repl.Replace(clean)
	parts := strings.Fields(clean)
	seen := map[string]struct{}{}
	out := []string{}
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || len([]rune(p)) < 2 {
			continue
		}
		upper := strings.ToUpper(p)
		if strings.Contains(upper, "腾讯") || strings.Contains(upper, "小米") || strings.Contains(upper, "美团") {
			continue
		}
		if strings.Contains(p, "策略") && len([]rune(p)) <= 4 {
			continue
		}
		if strings.Contains(p, "挨个") || strings.Contains(p, "依次") || strings.HasPrefix(p, "你") {
			continue
		}
		key := strings.ToLower(p)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, p)
		if len(out) >= maxStrategies {
			break
		}
	}
	return out
}

func applyStrategyFrequency(item *StrategyItem) {
	if item == nil {
		return
	}
	q := strings.ToUpper(item.Query + " " + item.Label)
	switch {
	case strings.Contains(q, "4H") || strings.Contains(item.Query, "节奏"):
		item.Frequency = "60m"
	case strings.Contains(item.Query, "共振"):
		item.Frequency = "15m"
	}
	if item.MonthsBack <= 0 {
		item.MonthsBack = defaultMonthsBack
	}
}

func defaultCatalogStrategies(items []map[string]any, skipLabel string) []StrategyItem {
	if len(items) == 0 {
		return nil
	}
	prefer := []string{"Macd4H", "共振", "SAR", "MACD"}
	out := []StrategyItem{}
	seen := map[string]struct{}{}
	skip := strings.ToLower(strings.TrimSpace(skipLabel))
	add := func(name string) {
		key := strings.ToLower(strings.TrimSpace(name))
		if key == "" || key == skip {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		item := StrategyItem{
			Query:      name,
			Kind:       "combination",
			Label:      name,
			Status:     StepPending,
			MonthsBack: defaultMonthsBack,
		}
		applyStrategyFrequency(&item)
		out = append(out, item)
	}
	for _, token := range prefer {
		for _, row := range items {
			name := strings.TrimSpace(fmt.Sprint(row["name"]))
			upper := strings.ToUpper(name)
			if strings.Contains(upper, strings.ToUpper(token)) {
				add(name)
			}
		}
	}
	for _, row := range items {
		if len(out) >= maxStrategies {
			break
		}
		add(fmt.Sprint(row["name"]))
	}
	if len(out) > maxStrategies {
		out = out[:maxStrategies]
	}
	return out
}

func (r *Runner) advanceMultiStrategy(ctx context.Context, session *runtime.Session, flow *Flow, toolCtx tools.Context, recordTool func(name, status, summary string)) error {
	switch flow.Phase {
	case PhaseResolveSymbol:
		return r.phaseResolveSymbol(ctx, session, flow, toolCtx, recordTool)
	case PhasePickStrategies:
		return r.phasePickStrategies(ctx, session, flow, toolCtx, recordTool)
	case PhaseProbeForeach:
		return r.phaseProbeForeach(ctx, flow, toolCtx, recordTool)
	case PhaseSummarize, PhaseDone:
		return nil
	default:
		flow.Phase = PhaseResolveSymbol
		return r.phaseResolveSymbol(ctx, session, flow, toolCtx, recordTool)
	}
}

func (r *Runner) phaseResolveSymbol(ctx context.Context, session *runtime.Session, flow *Flow, toolCtx tools.Context, recordTool func(name, status, summary string)) error {
	if strings.TrimSpace(flow.StockCode) != "" {
		flow.Phase = PhasePickStrategies
		return nil
	}
	query := strings.TrimSpace(flow.StockQuery)
	if query == "" {
		st := sessiontask.BuildState(session, "signal_probe")
		query = strings.TrimSpace(st.Symbol)
	}
	if query == "" {
		return terminalError("缺少标的，请说明要测哪只股票")
	}
	code, name, _, err := slots.ResolveStock(ctx, toolCtx, r.runToolCall(recordTool), query)
	if err != nil {
		return err
	}
	flow.StockQuery = query
	flow.StockCode = code
	flow.StockName = name
	flow.Phase = PhasePickStrategies
	flow.touch()
	return nil
}

func (r *Runner) phasePickStrategies(ctx context.Context, session *runtime.Session, flow *Flow, toolCtx tools.Context, recordTool func(name, status, summary string)) error {
	if len(flow.Strategies) > 0 {
		flow.Phase = PhaseProbeForeach
		flow.touch()
		return nil
	}
	res := r.runTool(ctx, toolCtx, "get_signal_combinations", map[string]any{}, recordTool)
	if res.Status != tools.StatusOK {
		return fmt.Errorf("get_signal_combinations 失败：%s", res.Summary)
	}
	items := slots.CatalogItems(res.Data)
	st := sessiontask.BuildState(session, "signal_probe")
	flow.Strategies = defaultCatalogStrategies(items, st.Strategy)
	if len(flow.Strategies) == 0 {
		return terminalError("没有可用的组合策略")
	}
	flow.Phase = PhaseProbeForeach
	flow.touch()
	return nil
}

func (r *Runner) phaseProbeForeach(ctx context.Context, flow *Flow, toolCtx tools.Context, recordTool func(name, status, summary string)) error {
	for i := range flow.Strategies {
		item := &flow.Strategies[i]
		if item.Status == StepDone || item.Status == StepSkipped {
			continue
		}
		if item.Status == StepFailed && !r.retryFailedOnly {
			continue
		}
		item.Status = StepRunning
		r.emitCard(flow)
		var lastErr error
		for attempt := 0; attempt <= maxStepRetries; attempt++ {
			item.Attempts = attempt + 1
			err := r.probeOneStrategy(ctx, flow, item, toolCtx, recordTool)
			if err == nil {
				item.Status = StepDone
				item.LastError = ""
				flow.Cursor = i + 1
				flow.PartialReport = renderPartialReport(flow)
				flow.touch()
				r.emitCard(flow)
				lastErr = nil
				break
			}
			lastErr = err
			item.LastError = err.Error()
		}
		if lastErr != nil {
			item.Status = StepSkipped
			flow.PartialReport = renderPartialReport(flow)
			flow.touch()
			r.emitCard(flow)
			if r.OnProgress != nil {
				r.OnProgress("workflow_step_skipped", map[string]any{
					"strategy": item.Label,
					"error":    lastErr.Error(),
				})
			}
		}
	}
	flow.Phase = PhaseSummarize
	return nil
}

func (r *Runner) probeOneStrategy(ctx context.Context, flow *Flow, item *StrategyItem, toolCtx tools.Context, recordTool func(name, status, summary string)) error {
	plan := slots.SignalPlan{SignalQuery: item.Query, SignalKind: item.Kind}
	sig, err := slots.ResolveSignal(ctx, toolCtx, r.runToolCall(recordTool), plan)
	if err != nil {
		return err
	}
	item.Label = sig.StrategyLabel
	item.Query = sig.StrategyLabel
	frequency := sig.Frequency
	if strings.TrimSpace(item.Frequency) != "" {
		frequency = item.Frequency
	}
	months := item.MonthsBack
	if months <= 0 {
		months = flow.MonthsBack
	}
	if months <= 0 {
		months = defaultMonthsBack
	}
	applyStrategyFrequency(item)
	if strings.TrimSpace(item.Frequency) != "" {
		frequency = item.Frequency
	}
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
	item.BuyHits = intAny(res.Data["buy_hits"])
	item.SellHits = intAny(res.Data["sell_hits"])
	item.Summary = strings.TrimSpace(res.Summary)
	item.Frequency = frequency
	item.MonthsBack = months
	return nil
}

func renderPartialReport(flow *Flow) string {
	if flow == nil {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "## %s %s · 多策略信号对比（进行中）\n\n", flow.StockName, flow.StockCode)
	fmt.Fprintf(&b, "| 策略 | 周期 | 买次 | 卖次 | 状态 |\n")
	fmt.Fprintf(&b, "| --- | --- | ---: | ---: | --- |\n")
	for _, item := range flow.Strategies {
		status := item.Status
		if status == "" {
			status = StepPending
		}
		buy := "-"
		sell := "-"
		if item.Status == StepDone {
			buy = fmt.Sprintf("%d", item.BuyHits)
			sell = fmt.Sprintf("%d", item.SellHits)
		}
		freq := item.Frequency
		if freq == "" {
			freq = "-"
		}
		label := item.Label
		if label == "" {
			label = item.Query
		}
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", label, freq, buy, sell, status)
	}
	return strings.TrimSpace(b.String())
}

func renderFinalReport(flow *Flow) string {
	body := renderPartialReport(flow)
	body = strings.Replace(body, "（进行中）", "（完成）", 1)
	done, skipped, failed := 0, 0, 0
	for _, item := range flow.Strategies {
		switch item.Status {
		case StepDone:
			done++
		case StepSkipped:
			skipped++
		case StepFailed:
			failed++
		}
	}
	footer := fmt.Sprintf("\n\n> 共 %d 个策略：成功 %d，跳过 %d，失败 %d。由 workflow 串行执行（probe_bot_signal_series）。",
		len(flow.Strategies), done, skipped, failed)
	return body + footer
}

func terminalError(msg string) error {
	return fmt.Errorf("%s", msg)
}

func intAny(v any) int {
	switch t := v.(type) {
	case int:
		return t
	case int64:
		return int(t)
	case float64:
		return int(t)
	default:
		if s := strings.TrimSpace(fmt.Sprint(v)); s != "" && s != "<nil>" {
			var n int
			_, _ = fmt.Sscanf(s, "%d", &n)
			return n
		}
	}
	return 0
}

func newRunID() string {
	return "flow-" + time.Now().UTC().Format("20060102T%H%M%S")
}
