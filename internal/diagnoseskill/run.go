package diagnoseskill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const DefaultMonthsBack = 3

// RunInput configures one diagnose skill execution.
type RunInput struct {
	Macro       bool
	Code        string
	SignalFreq  string
	AsOf        string
	MonthsBack  int
	SessionID   string
	RunID       string
	Lanes       *LaneClient
	JEV         *strategydev.JevClient
}

// Result is the skill `result` object (slim contract).
type Result map[string]any

// Run fetches 3220 lanes, runs four-lane JEV + unified, returns envelope fields.
func Run(ctx context.Context, in RunInput) (Result, error) {
	code := strings.TrimSpace(in.Code)
	if code == "" {
		return nil, fmt.Errorf("code required")
	}
	if !in.Macro && strings.TrimSpace(in.SignalFreq) == "" {
		return nil, fmt.Errorf("signal_frequency required for diagnose_strategy")
	}
	months := in.MonthsBack
	if months <= 0 {
		months = DefaultMonthsBack
	}
	asOf := strings.TrimSpace(in.AsOf)
	if in.Lanes == nil {
		return nil, fmt.Errorf("macro-api client not configured")
	}
	var bundle LaneBundle
	var err error
	if in.Macro {
		bundle, err = in.Lanes.FetchMacroLanes(ctx, code, asOf, months)
	} else {
		bundle, err = in.Lanes.FetchStrategyLanes(ctx, code, in.SignalFreq, asOf, months)
	}
	if err != nil {
		return nil, err
	}

	detail, anchorTime, anchorClose, _, err := populateDetailFromLanes(bundle, in.Macro)
	if err != nil {
		return nil, err
	}
	signalFreq := strings.TrimSpace(in.SignalFreq)
	if in.Macro {
		signalFreq = "daily"
	}
	jevCtx, cancel := context.WithTimeout(ctx, strategydev.DefaultLabDetailJEVTimeout)
	defer cancel()

	if detail.PriceAnalysis != nil && strategydev.LabPAInformative(detail.PriceAnalysis) {
		rec := strategydev.ResolvePriceAnalysisJEV(jevCtx, in.JEV, detail.PriceAnalysis, "", false)
		detail.PriceAnalysisJEV = &rec
	}
	if kline := strategydev.LabJEVKlineInputFromDetail(detail); kline != nil && !strategydev.LabJEVKlineInputIsEmpty(kline) {
		rec := strategydev.ResolveKlineJEV(jevCtx, in.JEV, kline, "", false)
		detail.KlineJEV = &rec
	}
	if detail.TechnicalAnalysis != nil {
		rec := strategydev.ResolveTechnicalAnalysisJEV(jevCtx, in.JEV, detail.TechnicalAnalysis, "", signalFreq, false)
		detail.TechnicalAnalysisJEV = &rec
	}
	if detail.IndexTrend != nil {
		rec := strategydev.ResolveIndexTrendJEV(jevCtx, in.JEV, detail.IndexTrend, "", signalFreq, false)
		detail.IndexTrendJEV = &rec
	}
	profile := strings.TrimSpace(detail.KlineJEVProfile)
	strategydev.ApplyDiagnoseJEVUnified(jevCtx, detail, in.JEV, signalFreq, profile)

	res := Result{
		"code": code,
		"time": anchorTime,
		"close": anchorClose,
		"price_analysis":      jsonArray(bundle.PriceAnalysis, "price_analysis"),
		"kline_pattern":       jsonArray(bundle.KlinePattern, "kline_pattern"),
		"technical_analysis":  jsonArray(bundle.TechnicalAnalysis, "technical_analysis"),
		"index_trend":         jsonArray(bundle.IndexTrend, "index_trend"),
	}
	if !in.Macro {
		res["signal_frequency"] = strings.TrimSpace(in.SignalFreq)
	} else if monitorClose := macroMonitorSessionClose(code, anchorTime); monitorClose != "" {
		res["monitor_session_close"] = monitorClose
	}
	if detail.PriceAnalysisJEV != nil {
		res["price_analysis_jev"] = emitLaneJEV(*detail.PriceAnalysisJEV)
	}
	if detail.KlineJEV != nil {
		res["kline_jev"] = emitKlineJEV(*detail.KlineJEV)
	}
	if detail.TechnicalAnalysisJEV != nil {
		res["technical_analysis_jev"] = emitTAJEV(*detail.TechnicalAnalysisJEV)
	}
	if detail.IndexTrendJEV != nil {
		res["index_trend_jev"] = emitIndexJEV(*detail.IndexTrendJEV)
	}
	res["jev_unified"] = emitUnifiedJEV(detail.JEVUnified)
	return res, nil
}

func jsonArray(raw json.RawMessage, key string) any {
	arr := rawLaneArray(raw, key)
	if len(arr) == 0 {
		return []any{}
	}
	var out any
	if json.Unmarshal(arr, &out) == nil {
		return out
	}
	return []any{}
}

func macroMonitorSessionClose(code, anchorTime string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	spec, ok := strategydev.MacroSessionSpecForExport(code)
	if !ok {
		return ""
	}
	// Use anchor calendar day when present; else today in exchange zone.
	day := time.Now()
	if anchorTime != "" {
		if i := strings.Index(anchorTime, " "); i > 0 {
			if t, err := time.Parse("2006-01-02", anchorTime[:i]); err == nil {
				day = t
			}
		}
	}
	closeAt := strategydev.MacroSessionCloseInstantExport(day, spec)
	if closeAt.IsZero() {
		return ""
	}
	return closeAt.UTC().Format(time.RFC3339)
}