package chat

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const (
	MetaKeySignalDiagnoseResults       = "signal_diagnose_results"
	MetaKeySignalDiagnoseResultLatest  = "signal_diagnose_result_latest"
	signalDiagnoseResultTTL            = 48 * time.Hour
	maxSignalDiagnoseResults           = 24
)

// SignalDiagnoseResult is the SSOT payload Monday polls after a chat turn (skill stepwise or runner).
type SignalDiagnoseResult struct {
	ResultKey     string `json:"result_key"`
	SessionID     string `json:"session_id,omitempty"`
	RunID         string `json:"run_id,omitempty"`
	StockCode     string `json:"stock_code,omitempty"`
	StrategyQuery string `json:"strategy_query,omitempty"`
	CatalogLabel  string `json:"catalog_label,omitempty"`
	Frequency     string `json:"frequency,omitempty"`
	MonthsBack    int    `json:"months_back,omitempty"`

	ChartProbe    map[string]any              `json:"chart_probe,omitempty"`
	SignalEval    slots.SignalEpisodeEval     `json:"signal_eval,omitempty"`
	DiagnoseDetail map[string]any             `json:"diagnose_detail,omitempty"`
	EvalJudgment  string                      `json:"eval_judgment,omitempty"`

	LabMode               bool                                      `json:"lab_mode,omitempty"`
	TradeDirection        runtime.TradeDirection                    `json:"trade_direction,omitempty"`
	LabEpisodeSummaries   strategydev.LabCardEpisodeSummaries       `json:"lab_episode_summaries,omitempty"`
	LabFlowPriceSummary   string                                    `json:"lab_flow_price_summary,omitempty"`
	LabDefaultDetailRef   string                                    `json:"lab_default_detail_ref,omitempty"`

	PriceAnalysis             *strategydev.PriceAnalysis             `json:"price_analysis,omitempty"`
	PriceEpisodeFilterStats   *strategydev.EpisodeTriggerFilterStats `json:"price_episode_filter_stats,omitempty"`
	UsePriceAnalysisEpisodeFilter bool                               `json:"use_price_analysis_episode_filter,omitempty"`

	CardStage  string    `json:"card_stage,omitempty"`
	Status     string    `json:"status,omitempty"`
	Phase      string    `json:"phase,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func newSignalDiagnoseResultKey() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "sd-res-" + hex.EncodeToString(b[:])
}

func pruneSignalDiagnoseResults(m map[string]json.RawMessage) {
	if len(m) == 0 {
		return
	}
	now := time.Now().UTC()
	for id, raw := range m {
		var ent SignalDiagnoseResult
		if err := json.Unmarshal(raw, &ent); err != nil || !ent.ExpiresAt.After(now) {
			delete(m, id)
		}
	}
	for len(m) > maxSignalDiagnoseResults {
		for id := range m {
			delete(m, id)
			break
		}
	}
}

func loadSignalDiagnoseResultsMap(meta map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if meta == nil {
		return out
	}
	raw, ok := meta[MetaKeySignalDiagnoseResults]
	if !ok || raw == nil {
		return out
	}
	switch v := raw.(type) {
	case map[string]json.RawMessage:
		for k, val := range v {
			out[k] = val
		}
	case map[string]any:
		for k, val := range v {
			switch t := val.(type) {
			case json.RawMessage:
				out[k] = t
			default:
				b, err := json.Marshal(t)
				if err == nil {
					out[k] = b
				}
			}
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return out
		}
		var generic map[string]json.RawMessage
		if err := json.Unmarshal(b, &generic); err == nil {
			return generic
		}
	}
	return out
}

func syncSignalDiagnoseResultsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	if len(rt.SignalDiagnoseResults) > 0 {
		merged := loadSignalDiagnoseResultsMap(chat.Metadata)
		for id, raw := range rt.SignalDiagnoseResults {
			merged[id] = raw
		}
		pruneSignalDiagnoseResults(merged)
		chat.Metadata[MetaKeySignalDiagnoseResults] = merged
	}
	if k := strings.TrimSpace(rt.SignalDiagnoseResultLatest); k != "" {
		chat.Metadata[MetaKeySignalDiagnoseResultLatest] = k
	}
}

func loadSignalDiagnoseResultsIntoSession(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	merged := loadSignalDiagnoseResultsMap(chat.Metadata)
	if len(merged) > 0 {
		if rt.SignalDiagnoseResults == nil {
			rt.SignalDiagnoseResults = map[string]json.RawMessage{}
		}
		for id, raw := range merged {
			rt.SignalDiagnoseResults[id] = raw
		}
	}
	if v, ok := chat.Metadata[MetaKeySignalDiagnoseResultLatest].(string); ok {
		rt.SignalDiagnoseResultLatest = strings.TrimSpace(v)
	}
}

// SignalDiagnoseResultMondayResponse is the POST /v1/signal-diagnose/result body (trading_operation PR #57).
type SignalDiagnoseResultMondayResponse struct {
	ChartProbe   map[string]any          `json:"chart_probe,omitempty"`
	SignalEval   slots.SignalEpisodeEval `json:"signal_eval,omitempty"`
	EvalJudgment string                  `json:"eval_judgment,omitempty"`
	Lab          any                     `json:"lab,omitempty"`
}

// MondayAPIResponse builds the Monday client field set (no ui-payload fallback).
func MondayAPIResponse(ent *SignalDiagnoseResult) SignalDiagnoseResultMondayResponse {
	if ent == nil {
		return SignalDiagnoseResultMondayResponse{}
	}
	out := SignalDiagnoseResultMondayResponse{
		ChartProbe: ent.ChartProbe,
		SignalEval: ent.SignalEval,
	}
	if s := strings.TrimSpace(ent.EvalJudgment); s != "" {
		out.EvalJudgment = s
	}
	labSums := strategydev.FilterLabCardSummariesForLongOnly(ent.LabEpisodeSummaries, ent.TradeDirection.LongOnlyEpisodeEval())
	if len(labSums) > 0 {
		out.Lab = labSums
	} else if strings.TrimSpace(ent.LabFlowPriceSummary) != "" || strings.TrimSpace(ent.LabDefaultDetailRef) != "" {
		lab := map[string]any{}
		if len(labSums) > 0 {
			lab["lab_episode_summaries"] = labSums
		}
		if s := strings.TrimSpace(ent.LabFlowPriceSummary); s != "" {
			lab["lab_flow_price_summary"] = s
		}
		if s := strings.TrimSpace(ent.LabDefaultDetailRef); s != "" {
			lab["lab_default_detail_ref"] = s
		}
		if len(lab) > 0 {
			out.Lab = lab
		}
	}
	return out
}

// LookupSignalDiagnoseResultForMonday returns the latest result, or the newest matching run_id when set.
func LookupSignalDiagnoseResultForMonday(chat *chatsession.ChatSession, runID string) (*SignalDiagnoseResult, bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return LookupSignalDiagnoseResult(chat, "")
	}
	if ent, ok := lookupSignalDiagnoseResultByRunOrKey(chat, runID); ok {
		return ent, true
	}
	return fallbackSignalDiagnoseResultWhenRunIDUnknown(chat, runID)
}

// LookupSignalDiagnoseResultForLabDetail resolves Persist Result for lab-detail (same as Monday result, plus single-result fallback).
func LookupSignalDiagnoseResultForLabDetail(chat *chatsession.ChatSession, runID string) (*SignalDiagnoseResult, bool) {
	runID = strings.TrimSpace(runID)
	if runID == "" {
		return LookupSignalDiagnoseResult(chat, "")
	}
	if ent, ok := lookupSignalDiagnoseResultByRunOrKey(chat, runID); ok {
		return ent, true
	}
	return fallbackSignalDiagnoseResultWhenRunIDUnknown(chat, runID)
}

func lookupSignalDiagnoseResultByRunOrKey(chat *chatsession.ChatSession, runID string) (*SignalDiagnoseResult, bool) {
	if chat == nil || runID == "" {
		return nil, false
	}
	m := loadSignalDiagnoseResultsMap(chat.Metadata)
	var best *SignalDiagnoseResult
	for key, raw := range m {
		var ent SignalDiagnoseResult
		if err := json.Unmarshal(raw, &ent); err != nil {
			continue
		}
		if time.Now().UTC().After(ent.ExpiresAt) {
			continue
		}
		if strings.TrimSpace(ent.RunID) != runID && key != runID && strings.TrimSpace(ent.ResultKey) != runID {
			continue
		}
		if best == nil || ent.UpdatedAt.After(best.UpdatedAt) {
			copy := ent
			best = &copy
		}
	}
	if best != nil {
		return best, true
	}
	return nil, false
}

// fallbackSignalDiagnoseResultWhenRunIDUnknown handles Monday client run ids that are not agent workflow run_id.
func fallbackSignalDiagnoseResultWhenRunIDUnknown(chat *chatsession.ChatSession, runID string) (*SignalDiagnoseResult, bool) {
	if chat == nil || runID == "" {
		return nil, false
	}
	m := loadSignalDiagnoseResultsMap(chat.Metadata)
	var only *SignalDiagnoseResult
	count := 0
	for _, raw := range m {
		var ent SignalDiagnoseResult
		if err := json.Unmarshal(raw, &ent); err != nil {
			continue
		}
		if time.Now().UTC().After(ent.ExpiresAt) || ent.ChartProbe == nil {
			continue
		}
		count++
		copy := ent
		only = &copy
	}
	if count == 1 && only != nil {
		return only, true
	}
	return LookupSignalDiagnoseResult(chat, "")
}

// LookupSignalDiagnoseResult loads a stored result by key; empty key returns the latest for the session.
func LookupSignalDiagnoseResult(chat *chatsession.ChatSession, key string) (*SignalDiagnoseResult, bool) {
	if chat == nil {
		return nil, false
	}
	key = strings.TrimSpace(key)
	if key == "" {
		if latest, ok := chat.Metadata[MetaKeySignalDiagnoseResultLatest].(string); ok {
			key = strings.TrimSpace(latest)
		}
	}
	if key == "" {
		return nil, false
	}
	raw, ok := loadSignalDiagnoseResultsMap(chat.Metadata)[key]
	if !ok || len(raw) == 0 {
		return nil, false
	}
	var ent SignalDiagnoseResult
	if err := json.Unmarshal(raw, &ent); err != nil {
		return nil, false
	}
	if time.Now().UTC().After(ent.ExpiresAt) {
		return nil, false
	}
	return &ent, true
}

func storeSignalDiagnoseResult(rt *runtime.Session, chat *chatsession.ChatSession, ent SignalDiagnoseResult) string {
	if ent.ChartProbe == nil {
		return ""
	}
	now := time.Now().UTC()
	key := strings.TrimSpace(ent.ResultKey)
	if key == "" {
		key = newSignalDiagnoseResultKey()
	}
	ent.ResultKey = key
	if ent.CreatedAt.IsZero() {
		ent.CreatedAt = now
	}
	ent.UpdatedAt = now
	ent.ExpiresAt = now.Add(signalDiagnoseResultTTL)
	if ent.SessionID == "" {
		ent.SessionID = sessionIDFromStore(rt, chat)
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		return ""
	}
	if rt != nil {
		if rt.SignalDiagnoseResults == nil {
			rt.SignalDiagnoseResults = map[string]json.RawMessage{}
		}
		rt.SignalDiagnoseResults[key] = raw
		rt.SignalDiagnoseResultLatest = key
		if rt.SignalDiagnoseUIPending != nil {
			rt.SignalDiagnoseUIPending.ResultKey = key
		}
	}
	if chat != nil {
		if chat.Metadata == nil {
			chat.Metadata = map[string]any{}
		}
		merged := loadSignalDiagnoseResultsMap(chat.Metadata)
		merged[key] = raw
		pruneSignalDiagnoseResults(merged)
		chat.Metadata[MetaKeySignalDiagnoseResults] = merged
		chat.Metadata[MetaKeySignalDiagnoseResultLatest] = key
	}
	return key
}

func mergeResultFromExisting(key string, chat *chatsession.ChatSession, rt *runtime.Session) SignalDiagnoseResult {
	if key != "" {
		if ent, ok := LookupSignalDiagnoseResult(chat, key); ok && ent != nil {
			return *ent
		}
		if rt != nil && rt.SignalDiagnoseResults != nil {
			if raw, ok := rt.SignalDiagnoseResults[key]; ok && len(raw) > 0 {
				var ent SignalDiagnoseResult
				if err := json.Unmarshal(raw, &ent); err == nil {
					return ent
				}
			}
		}
	}
	return SignalDiagnoseResult{ResultKey: key}
}

func resultFromFlow(rt *runtime.Session, chat *chatsession.ChatSession, flow *Flow) SignalDiagnoseResult {
	key := ""
	if flow != nil && strings.TrimSpace(flow.DiagnoseUIKey) != "" {
		key = strings.TrimSpace(flow.DiagnoseUIKey)
	}
	if key == "" && rt != nil && rt.SignalDiagnoseUIPending != nil {
		key = strings.TrimSpace(rt.SignalDiagnoseUIPending.ResultKey)
	}
	ent := mergeResultFromExisting(key, chat, rt)
	if flow == nil {
		return ent
	}
	cp, eval, ok := uiPayloadFromFlow(flow)
	if ok {
		ent.ChartProbe = cp
		ent.SignalEval = eval
	}
	ensureDiagnoseRunIDOnFlow(flow, chat, rt)
	if flow.RunID != "" {
		ent.RunID = flow.RunID
	}
	if flow.StockCode != "" {
		ent.StockCode = flow.StockCode
	}
	if flow.StrategyQuery != "" {
		ent.StrategyQuery = flow.StrategyQuery
	}
	if flow.CatalogLabel != "" {
		ent.CatalogLabel = flow.CatalogLabel
	}
	ent.Frequency = probeFrequency(flow)
	ent.MonthsBack = flow.MonthsBack
	if d := diagnoseStructuredForCard(flow.DiagnoseRaw); d != nil {
		ent.DiagnoseDetail = d
	}
	if s := strings.TrimSpace(flow.EvalJudgment); s != "" {
		ent.EvalJudgment = s
	}
	ent.LabMode = flow.LabMode
	if td := strings.TrimSpace(string(flow.TradeDirection)); td != "" {
		ent.TradeDirection = flow.TradeDirection
	} else {
		ent.TradeDirection = runtime.TradeDirectionBoth
	}
	if s := strings.TrimSpace(flow.LabFlowPriceSummary); s != "" {
		ent.LabFlowPriceSummary = s
	}
	if flow.ProbeRaw != nil {
		if len(flow.LabEpisodeSummaries) == 0 {
			populateFlowLabSummariesFromProbe(flow)
		} else {
			flow.LabEpisodeSummaries = strategydev.FilterLabCardSummariesForLongOnly(
				flow.LabEpisodeSummaries,
				flow.TradeDirection.LongOnlyEpisodeEval(),
			)
			ensureLabFlowDetailRef(flow)
		}
		longOnly := flow.TradeDirection.LongOnlyEpisodeEval()
		if len(flow.LabEpisodeSummaries) > 0 {
			ent.LabEpisodeSummaries = strategydev.FilterLabCardSummariesForLongOnly(flow.LabEpisodeSummaries, longOnly)
		}
		ent.LabDefaultDetailRef = strategydev.LabFlowDetailRef
	} else if len(flow.LabEpisodeSummaries) > 0 {
		ent.LabEpisodeSummaries = strategydev.FilterLabCardSummariesForLongOnly(
			flow.LabEpisodeSummaries,
			flow.TradeDirection.LongOnlyEpisodeEval(),
		)
	}
	ent.UsePriceAnalysisEpisodeFilter = flow.UsePriceAnalysisEpisodeFilter
	ent.PriceAnalysis = flow.PriceAnalysis
	ent.PriceEpisodeFilterStats = flow.PriceEpisodeFilterStats
	ent.CardStage = flow.CardStage
	ent.Status = flow.Status
	ent.Phase = flow.Phase
	return ent
}

func draftFlowFromRuntime(rt *runtime.Session) *Flow {
	if rt == nil || rt.SignalDiagnoseUIPending == nil {
		return nil
	}
	p := rt.SignalDiagnoseUIPending
	if p.ProbeRaw == nil {
		return nil
	}
	flow := &Flow{
		Template:   SkillSignalDiagnose,
		ProbeRaw:   p.ProbeRaw,
		StockCode:  p.StockCode,
		MonthsBack: p.MonthsBack,
	}
	if flow.StockCode == "" {
		if code, _ := p.ProbeRaw["code"].(string); strings.TrimSpace(code) != "" {
			flow.StockCode = strings.TrimSpace(code)
		}
	}
	if p.Frequency != "" {
		flow.Strategies = []StrategyItem{{Frequency: p.Frequency}}
	} else if freq, _ := p.ProbeRaw["frequency"].(string); strings.TrimSpace(freq) != "" {
		flow.Strategies = []StrategyItem{{Frequency: strings.TrimSpace(freq)}}
	}
	if p.LongOnly {
		flow.TradeDirection = runtime.TradeDirectionLongOnly
	}
	if p.DiagnoseRaw != nil {
		flow.DiagnoseRaw = p.DiagnoseRaw
	}
	if p.StrategyQuery != "" {
		flow.StrategyQuery = p.StrategyQuery
	}
	ApplyMergedPanelOpsToFlowForPersist(rt, flow)
	populateFlowLabSummariesFromProbe(flow)
	return flow
}

// RememberSignalDiagnoseDiagnose archives diagnose_bot_signal_series payload for result persistence.
func RememberSignalDiagnoseDiagnose(rt *runtime.Session, diagnose map[string]any) {
	if rt == nil || len(diagnose) == 0 {
		return
	}
	cp := map[string]any{}
	for k, v := range diagnose {
		cp[k] = v
	}
	if rt.SignalDiagnoseUIPending == nil {
		rt.SignalDiagnoseUIPending = &runtime.SignalDiagnoseUIPending{}
	}
	rt.SignalDiagnoseUIPending.DiagnoseRaw = cp
}

func ensureDiagnoseRunIDOnFlow(flow *Flow, chat *chatsession.ChatSession, rt *runtime.Session) {
	if flow == nil || flow.RunID != "" {
		return
	}
	if ent, ok := LookupSignalDiagnoseResult(chat, ""); ok && ent != nil && strings.TrimSpace(ent.RunID) != "" {
		flow.RunID = strings.TrimSpace(ent.RunID)
		return
	}
	flow.RunID = newRunID()
}

// PersistSignalDiagnoseResultFromFlow writes or updates the session result from workflow or skill draft state.
func PersistSignalDiagnoseResultFromFlow(rt *runtime.Session, chat *chatsession.ChatSession, flow *Flow) string {
	if flow == nil {
		flow = draftFlowFromRuntime(rt)
	}
	if flow == nil {
		return ""
	}
	ensureDiagnoseRunIDOnFlow(flow, chat, rt)
	ent := resultFromFlow(rt, chat, flow)
	if ent.ChartProbe == nil {
		return ""
	}
	key := storeSignalDiagnoseResult(rt, chat, ent)
	if key != "" && flow != nil {
		flow.DiagnoseUIKey = key
		syncLabDiagnoseSnapshotAfterPersist(chat, rt, flow)
	}
	return key
}

func syncLabDiagnoseSnapshotAfterPersist(chat *chatsession.ChatSession, rt *runtime.Session, flow *Flow) {
	if flow == nil || !FlowSupportsLabDetail(flow) {
		return
	}
	if chat != nil {
		UpsertLabDiagnoseSnapshot(chat, flow)
	}
	if rt != nil {
		ArchiveLabDiagnoseFlow(rt, flow)
	}
}

// UpdateSignalDiagnoseResultAfterTool persists partial results after probe/diagnose tools (skill stepwise path).
func UpdateSignalDiagnoseResultAfterTool(rt *runtime.Session, chat *chatsession.ChatSession, toolName string, data map[string]any) {
	if rt == nil || len(data) == 0 {
		return
	}
	switch toolName {
	case "probe_bot_signal_series", "probe_bot_signal":
		RememberSignalDiagnoseProbe(rt, data)
	case "diagnose_bot_signal_series":
		RememberSignalDiagnoseDiagnose(rt, data)
	default:
		return
	}
	_ = PersistSignalDiagnoseResultFromFlow(rt, chat, nil)
}

// FinalizeSignalDiagnoseResultFromSession merges active runner flow fields (eval_judgment, lab) into the latest result.
func FinalizeSignalDiagnoseResultFromSession(rt *runtime.Session, chat *chatsession.ChatSession) {
	if rt == nil {
		return
	}
	flow := LoadFromSession(rt)
	if flow == nil || canonicalSkill(flow.Template) != SkillSignalDiagnose {
		flow = draftFlowFromRuntime(rt)
	}
	if flow == nil {
		return
	}
	_ = PersistSignalDiagnoseResultFromFlow(rt, chat, flow)
}
