package strategydev

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Deps wires tool execution and optional Jev client for Run.
type Deps struct {
	Run ToolRunner
	Jev *JevClient
	Now func() time.Time
}

// Run executes the six-step strategy development orchestration workflow.
func Run(ctx context.Context, in Input, deps Deps) (Result, error) {
	if deps.Run == nil {
		return Result{}, fmt.Errorf("缺少 probe/diagnose 工具执行器")
	}
	now := time.Now().UTC()
	if deps.Now != nil {
		now = deps.Now()
	}
	code := strings.TrimSpace(in.Code)
	freq := NormalizeFrequency(in.Frequency)
	if code == "" || freq == "" {
		return Result{}, fmt.Errorf("需要标的 code 与周期 frequency")
	}
	months := CapMonthsBack(in.MonthsBack)
	limit := BarLimitForFrequency(freq)

	out := Result{
		Code: code, Frequency: freq, BarLimit: limit, MonthsBack: months,
	}

	// Step 1 — fetch data
	probeRes := deps.Run(ctx, "probe_bot_signal_series", map[string]any{
		"code":        code,
		"frequency":   freq,
		"limit":       limit,
		"buy_signal":  regimeProbeBuySignal(),
		"sell_signal": []any{map[string]any{"index": "nosignal", "type": "", "param": map[string]any{}}},
	})
	if probeRes.Status != tools.StatusOK {
		return out, fmt.Errorf("取数据失败：%s", probeRes.Summary)
	}
	bars, _ := probeRes.Data["bars"].([]any)
	bars = filterCompleteBars(bars, now)
	rules := probeRules(probeRes.Data)
	series := alignSeries(bars, valueSeriesFromRules(rules))

	// Step 2 — regime
	var prev *regimeOutcome
	if len(adxSeries(series)) > 1 {
		p := ClassifyRegime(bars[:len(bars)-1], series, nil)
		prev = &p
	}
	ro := ClassifyRegime(bars, series, prev)
	out.Regime = ro.Regime
	out.Side = ro.Side
	if out.Side == "" {
		out.Side = sideForRegime(out.Regime)
	}
	out.Confidence = ro.Confidence
	out.RegimeNote = formatRegimeNote(ro)

	if !regimeAllowsSelection(out.Regime) {
		out.Skipped = true
		out.SkipReason = "横盘或不明确，不选策略"
		out.FinalVerdict = VerdictSwapPair
		return out, nil
	}

	cat, err := LoadCatalog(ctx, deps.Run)
	if err != nil {
		return out, err
	}

	// Step 3 — filter
	out.BuyCandidates, out.SellCandidates = FilterCandidates(cat, out.Regime, out.Side, freq)
	if len(out.SellCandidates) == 0 {
		return out, fmt.Errorf("无匹配卖出候选")
	}
	if len(out.BuyCandidates) == 0 {
		return out, fmt.Errorf("无匹配买入候选")
	}

	// Step 4 — Jev pick
	jev := deps.Jev
	if jev == nil {
		jev = NewJevClient(JevConfig{})
	}
	buyID, sellID, pickReason, err := jev.PickStrategies(ctx, out.Regime, out.Side, freq, out.BuyCandidates, out.SellCandidates)
	if err != nil {
		return out, err
	}
	buyCand := findCandidate(out.BuyCandidates, buyID)
	sellCand := findCandidate(out.SellCandidates, sellID)
	if buyCand == nil {
		buyCand = &out.BuyCandidates[0]
	}
	if sellCand == nil {
		sellCand = &out.SellCandidates[0]
	}
	if isNoSignalRules(buyCand.Buy) {
		note, replaced := ensureBearCoverBuy(&cat, freq, out.BuyCandidates)
		if replaced != nil {
			buyCand = replaced
			out.BuyFallbackNote = note
		}
	}
	if isNoSignalRules(buyCand.Buy) {
		return out, fmt.Errorf("下跌时买入不能是 nosignal")
	}
	out.SelectedBuy = buyCand
	out.SelectedSell = sellCand
	_ = pickReason

	// Step 5 — diagnose (optional price-analysis trigger filter before episode metrics)
	usePriceFilter := UsePriceAnalysisEpisodeFilterEnabled(in.UsePriceAnalysisEpisodeFilter)
	buyScoreDir := ParseBuyScoreDirection(in.PriceAnalysisBuyScoreDirection)
	diag, pf, pfStats, pa, _, err := runDiagnose(ctx, deps.Run, code, freq, months, buyCand, sellCand, out.Regime, usePriceFilter, in.PriceAnalysisScoreThreshold, buyScoreDir)
	if err != nil {
		return out, err
	}
	out.Diagnose = diag
	out.PriceFilter = pf
	out.PriceEpisodeFilterStats = pfStats
	out.PriceAnalysis = pa

	// Step 6 — verdict
	jevVerdict, reason, _ := jev.Verdict(ctx, jevVerdictRequest{
		Task:         "strategy_dev_verdict",
		Diagnose:     diag,
		SelectedBuy:  buyCand.Name,
		SelectedSell: sellCand.Name,
	})
	out.JevVerdict = jevVerdict
	out.FinalVerdict = ApplyHardVerdictRules(diag)
	if out.FinalVerdict == VerdictPass && jevVerdict != VerdictPass {
		out.FinalVerdict = jevVerdict
	}
	if out.FinalVerdict == VerdictPass && jevVerdict == VerdictLongWindow {
		out.FinalVerdict = VerdictLongWindow
	}
	out.VerdictReason = reason
	if out.FinalVerdict != VerdictPass {
		out.VerdictReason = strings.TrimSpace(out.VerdictReason + " " + ApplyHardVerdictRules(diag))
	}
	return out, nil
}

func adxSeries(series map[string][]float64) []float64 {
	return pickSeries(series, "ADX", "ADX_ADX")
}

func findCandidate(list []Candidate, id string) *Candidate {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	for i := range list {
		if list[i].SignalID == id {
			cp := list[i]
			return &cp
		}
	}
	return nil
}

func ensureBearCoverBuy(cat *Catalog, freq string, existing []Candidate) (note string, c *Candidate) {
	if cat == nil {
		return "", nil
	}
	if x := cat.findByID(coverComboID); x != nil && !isNoSignalRules(x.Buy) {
		return "平空买入使用 SAR+MACD 直方图组合", x
	}
	for _, cand := range existing {
		blob := nameBlob(cand)
		if strings.Contains(blob, "吊灯信号") {
			return "平空买入退化为吊灯信号单指标", &cand
		}
	}
	for _, cand := range cat.Indices {
		blob := nameBlob(cand)
		if strings.Contains(blob, "SAR信号") || strings.Contains(blob, "SAR 信号") {
			return "平空买入退化为 SAR 信号单指标", &cand
		}
	}
	return "", nil
}
