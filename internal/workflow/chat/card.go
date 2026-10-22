package chat

import (
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

// CardPayload builds SSE card data for the live UI.
func CardPayload(flow *Flow) map[string]any {
	if flow == nil {
		return nil
	}
	strategies := make([]map[string]any, 0, len(flow.Strategies))
	done, total := 0, len(flow.Strategies)
	for _, s := range flow.Strategies {
		if s.Status == StepDone || s.Status == StepSkipped {
			done++
		}
		strategies = append(strategies, map[string]any{
			"query":      s.Query,
			"label":      s.Label,
			"frequency":  s.Frequency,
			"status":     s.Status,
			"buy_hits":   s.BuyHits,
			"sell_hits":  s.SellHits,
			"last_error": s.LastError,
		})
	}
	skill := flow.Template
	out := map[string]any{
		"run_id":         flow.RunID,
		"skill":          skill,
		"template":       skill, // legacy clients
		"template_name":  skillDisplayName(skill),
		"phase":          flow.Phase,
		"status":         flow.Status,
		"stock_query":    flow.StockQuery,
		"stock_code":     flow.StockCode,
		"stock_name":     flow.StockName,
		"months_back":    flow.MonthsBack,
		"cursor":         flow.Cursor,
		"total":          total,
		"done":           done,
		"strategies":     strategies,
		"partial_report": flow.PartialReport,
	}
	if canonicalSkill(skill) == SkillGenerateStrategyArchive || skill == SkillStrategyDev || skill == SkillStrategyDevPipeline {
		out["workflow_step"] = flow.WorkflowStep
		out["strategy_query"] = flow.StrategyQuery
		out["catalog_type"] = flow.CatalogType
		out["catalog_label"] = flow.CatalogLabel
		out["knowledge_id"] = flow.KnowledgeID
		out["kb_draft_chars"] = len(flow.KBDraft)
	}
	if canonicalSkill(skill) == SkillSignalDiagnose {
		out["strategy_query"] = flow.StrategyQuery
		out["catalog_type"] = flow.CatalogType
		out["catalog_label"] = flow.CatalogLabel
		out["knowledge_id"] = flow.KnowledgeID
		out["kb_draft_chars"] = len(flow.KBDraft)
		out["eval_judgment"] = flow.EvalJudgment
		if flow.ComposeTimeoutMs > 0 {
			out["compose_timeout_ms"] = flow.ComposeTimeoutMs
		}
		out["compose_fallback"] = flow.ComposeFallback
		if s := strings.TrimSpace(flow.ComposeModel); s != "" {
			out["compose_model"] = s
		}
		if s := strings.TrimSpace(flow.ComposeModelSource); s != "" {
			out["compose_model_source"] = s
		}
		if s := strings.TrimSpace(flow.ComposeModelSourceDetail); s != "" {
			out["compose_model_source_detail"] = s
		}
		if s := strings.TrimSpace(flow.ComposeReasoningDebug); s != "" {
			out["compose_reasoning_debug"] = s
		}
		if flow.ComposeReasoningChars > 0 {
			out["compose_reasoning_chars"] = flow.ComposeReasoningChars
		}
		if flow.ComposeFirstReasoningMs > 0 {
			out["compose_first_reasoning_ms"] = flow.ComposeFirstReasoningMs
		}
		if flow.ComposeFirstContentMs > 0 {
			out["compose_first_content_ms"] = flow.ComposeFirstContentMs
		}
		if flow.ComposeChunks > 0 {
			out["compose_chunks"] = flow.ComposeChunks
		}
		if flow.ComposeRequestTrace != nil {
			out["compose_request_trace"] = flow.ComposeRequestTrace
		}
		out["probe_buy_hits"] = flow.ProbeBuyHits
		out["probe_sell_hits"] = flow.ProbeSellHits
		out["probe_buy_hits_signal_api"] = flow.ProbeBuyHitsSignalAPI
		out["probe_sell_hits_signal_api"] = flow.ProbeSellHitsSignalAPI
		out["probe_bar_count"] = flow.ProbeBarCount
		out["probe_hits_scope"] = "in_window_after_warmup"
		if plan := probeSignalPlanForCard(flow); plan != nil {
			out["probe_signal_plan"] = plan
		}
		if d := diagnoseStructuredForCard(flow.DiagnoseRaw); d != nil {
			out["diagnose_detail"] = d
		}
		if td := strings.TrimSpace(string(flow.TradeDirection)); td != "" {
			out["trade_direction"] = td
		} else {
			out["trade_direction"] = string(runtime.TradeDirectionBoth)
		}
		if cp := chartProbePayload(flow); cp != nil {
			out["chart_probe"] = cp
		}
		if flow.SignalEval.Method != "" {
			out["signal_eval"] = flow.SignalEval
		}
		if strategydev.PriceAnalysisFeatureEnabled() {
			out["use_price_analysis_episode_filter"] = flow.UsePriceAnalysisEpisodeFilter
			if flow.PriceFilter != nil {
				out["price_filter"] = flow.PriceFilter
			}
			if flow.PriceAnalysis != nil {
				out["price_analysis"] = flow.PriceAnalysis
			}
		}
		if stage := strings.TrimSpace(flow.CardStage); stage != "" {
			out["card_stage"] = stage
		}
		if flow.LabMode {
			out["lab_mode"] = true
			ensureLabFlowDetailRef(flow)
			out["lab_default_detail_ref"] = strategydev.LabFlowDetailRef
			if len(flow.LabEpisodeSummaries) > 0 {
				out["lab_episode_summaries"] = flow.LabEpisodeSummaries
			}
			if s := strings.TrimSpace(flow.LabFlowPriceSummary); s != "" {
				out["lab_flow_price_summary"] = s
			}
		}
		if len(flow.PhaseTimingsMs) > 0 {
			out["phase_timings_ms"] = flow.PhaseTimingsMs
		}
		out["diagnose_merged_at_probe"] = flow.DiagnoseMergedAtProbe
		if flow.BuildDetailSkipped {
			out["build_detail_skipped"] = true
			if s := strings.TrimSpace(flow.BuildDetailSkipReason); s != "" {
				out["build_detail_skip_reason"] = s
			}
		}
		if flow.PriceEpisodeFilterStats != nil {
			if strategydev.PriceAnalysisFeatureEnabled() {
				out["price_episode_filter_stats"] = flow.PriceEpisodeFilterStats
			}
			if strategydev.KlinePatternDisplayEnabled() {
				if len(flow.PriceEpisodeFilterStats.BuyTriggers) > 0 {
					out["buy_triggers"] = flow.PriceEpisodeFilterStats.BuyTriggers
				}
				if len(flow.PriceEpisodeFilterStats.SellTriggers) > 0 {
					out["sell_triggers"] = flow.PriceEpisodeFilterStats.SellTriggers
				}
			}
		}
		if kl := keyLevelSnapshotToMap(flow.KeyLevels); kl != nil {
			out["key_levels"] = kl
		}
		if ser := keyLevelBarSeriesToMap(flow.KeyLevelSeries); ser != nil {
			out["key_level_series"] = ser
		}
	}
	return out
}

func skillDisplayName(id string) string {
	switch id {
	case "multi_strategy_compare":
		return "多策略信号对比"
	case "param_tune":
		return "策略参数调优"
	case "strategy_dev":
		return "策略开发"
	case SkillStrategyDevPipeline:
		return "策略开发编排"
	case SkillSignalDiagnose:
		return "信号诊断"
	case SkillGenerateStrategyArchive, legacyGenerateStrategyCognition:
		return "生成策略档案"
	default:
		return id
	}
}
