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
)

const (
	MetaKeySignalDiagnoseUIPayloads = "signal_diagnose_ui_payloads"
	signalDiagnoseUIPayloadTTL      = 48 * time.Hour
	maxSignalDiagnoseUIPayloads     = 24
)

// SignalDiagnoseUIEntry is persisted rich UI data for Monday lazy fetch.
type SignalDiagnoseUIEntry struct {
	ExpiresAt  time.Time               `json:"expires_at"`
	CreatedAt  time.Time               `json:"created_at"`
	RunID      string                  `json:"run_id,omitempty"`
	SessionID  string                  `json:"session_id,omitempty"`
	StockCode  string                  `json:"stock_code,omitempty"`
	ChartProbe map[string]any          `json:"chart_probe"`
	SignalEval slots.SignalEpisodeEval `json:"signal_eval"`
}

// RememberSignalDiagnoseProbe stores the full probe API payload on the runtime session for later UI archival.
func RememberSignalDiagnoseProbe(rt *runtime.Session, probe map[string]any) {
	if rt == nil || len(probe) == 0 {
		return
	}
	if _, ok := probe["bars"]; !ok {
		return
	}
	cp := map[string]any{}
	for k, v := range probe {
		cp[k] = v
	}
	rt.SignalDiagnoseUIPending = &runtime.SignalDiagnoseUIPending{ProbeRaw: cp}
	if code, _ := probe["code"].(string); strings.TrimSpace(code) != "" {
		rt.SignalDiagnoseUIPending.StockCode = strings.TrimSpace(code)
	}
	if freq, _ := probe["frequency"].(string); strings.TrimSpace(freq) != "" {
		rt.SignalDiagnoseUIPending.Frequency = strings.TrimSpace(freq)
	}
	if mb, ok := probe["months_back"].(float64); ok && mb > 0 {
		rt.SignalDiagnoseUIPending.MonthsBack = int(mb)
	}
}

func newSignalDiagnoseUIKey() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return "sd-ui-" + hex.EncodeToString(b[:])
}

func pruneSignalDiagnoseUIPayloads(m map[string]json.RawMessage) {
	if len(m) == 0 {
		return
	}
	now := time.Now().UTC()
	for id, raw := range m {
		var ent SignalDiagnoseUIEntry
		if err := json.Unmarshal(raw, &ent); err != nil || !ent.ExpiresAt.After(now) {
			delete(m, id)
		}
	}
	for len(m) > maxSignalDiagnoseUIPayloads {
		for id := range m {
			delete(m, id)
			break
		}
	}
}

func loadSignalDiagnoseUIMap(meta map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if meta == nil {
		return out
	}
	raw, ok := meta[MetaKeySignalDiagnoseUIPayloads]
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

func syncSignalDiagnoseUIPayloadsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || len(rt.SignalDiagnoseUIPayloads) == 0 {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := loadSignalDiagnoseUIMap(chat.Metadata)
	for id, raw := range rt.SignalDiagnoseUIPayloads {
		merged[id] = raw
	}
	pruneSignalDiagnoseUIPayloads(merged)
	chat.Metadata[MetaKeySignalDiagnoseUIPayloads] = merged
}

func loadSignalDiagnoseUIPayloadsIntoSession(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	merged := loadSignalDiagnoseUIMap(chat.Metadata)
	if len(merged) == 0 {
		return
	}
	if rt.SignalDiagnoseUIPayloads == nil {
		rt.SignalDiagnoseUIPayloads = map[string]json.RawMessage{}
	}
	for id, raw := range merged {
		rt.SignalDiagnoseUIPayloads[id] = raw
	}
}

// StoreSignalDiagnoseUIPayload persists rich chart_probe + signal_eval and returns a fetch key.
func StoreSignalDiagnoseUIPayload(rt *runtime.Session, chat *chatsession.ChatSession, ent SignalDiagnoseUIEntry) string {
	if ent.ChartProbe == nil {
		return ""
	}
	now := time.Now().UTC()
	ent.CreatedAt = now
	ent.ExpiresAt = now.Add(signalDiagnoseUIPayloadTTL)
	key := newSignalDiagnoseUIKey()
	raw, err := json.Marshal(ent)
	if err != nil {
		return ""
	}
	if rt != nil {
		if rt.SignalDiagnoseUIPayloads == nil {
			rt.SignalDiagnoseUIPayloads = map[string]json.RawMessage{}
		}
		rt.SignalDiagnoseUIPayloads[key] = raw
	}
	if chat != nil {
		if chat.Metadata == nil {
			chat.Metadata = map[string]any{}
		}
		merged := loadSignalDiagnoseUIMap(chat.Metadata)
		merged[key] = raw
		pruneSignalDiagnoseUIPayloads(merged)
		chat.Metadata[MetaKeySignalDiagnoseUIPayloads] = merged
	}
	return key
}

// LookupSignalDiagnoseUIPayload loads a stored payload by key from chat metadata.
func LookupSignalDiagnoseUIPayload(chat *chatsession.ChatSession, key string) (*SignalDiagnoseUIEntry, bool) {
	key = strings.TrimSpace(key)
	if chat == nil || key == "" {
		return nil, false
	}
	raw, ok := loadSignalDiagnoseUIMap(chat.Metadata)[key]
	if !ok || len(raw) == 0 {
		return nil, false
	}
	var ent SignalDiagnoseUIEntry
	if err := json.Unmarshal(raw, &ent); err != nil {
		return nil, false
	}
	if time.Now().UTC().After(ent.ExpiresAt) {
		return nil, false
	}
	return &ent, true
}

func uiPayloadFromFlow(flow *Flow) (chartProbe map[string]any, eval slots.SignalEpisodeEval, ok bool) {
	if flow == nil || flow.ProbeRaw == nil {
		return nil, eval, false
	}
	bars, _ := flow.ProbeRaw["bars"].([]any)
	if len(bars) == 0 {
		return nil, eval, false
	}
	cp := chartProbePayload(flow)
	if cp == nil {
		return nil, eval, false
	}
	eval = flow.SignalEval
	if eval.Method == "" {
		buyMerged, _ := flow.ProbeRaw["buy_merged"].([]any)
		sellMerged, _ := flow.ProbeRaw["sell_merged"].([]any)
		eval = slots.EvaluateSignalEpisodesWithOptions(bars, buyMerged, sellMerged, signalDiagnoseEpisodeEvalOpts(flow))
	}
	return cp, eval, true
}

// EnsureFlowDiagnoseUIKey stores rich payload for a workflow flow when ready and sets flow.DiagnoseUIKey.
func EnsureFlowDiagnoseUIKey(rt *runtime.Session, chat *chatsession.ChatSession, flow *Flow) string {
	if flow == nil {
		return ""
	}
	if k := strings.TrimSpace(flow.DiagnoseUIKey); k != "" {
		return k
	}
	cp, eval, ok := uiPayloadFromFlow(flow)
	if !ok {
		return ""
	}
	if flow.CardStage != "eval_done" && flow.Status != StatusCompleted && flow.Phase != PhaseDone {
		if eval.Method == "" {
			return ""
		}
	}
	key := StoreSignalDiagnoseUIPayload(rt, chat, SignalDiagnoseUIEntry{
		RunID: flow.RunID, SessionID: sessionIDFromStore(rt, chat),
		StockCode: flow.StockCode, ChartProbe: cp, SignalEval: eval,
	})
	if key != "" {
		flow.DiagnoseUIKey = key
	}
	return key
}

func sessionIDFromStore(rt *runtime.Session, chat *chatsession.ChatSession) string {
	if rt != nil && strings.TrimSpace(rt.ID) != "" {
		return rt.ID
	}
	if chat != nil {
		return chat.ID
	}
	return ""
}

func stripHeavySignalDiagnoseCard(out map[string]any) {
	delete(out, "signal_eval")
	if cp, ok := out["chart_probe"].(map[string]any); ok {
		delete(cp, "bars")
		delete(cp, "episodes")
		out["chart_probe"] = chartProbeLightFromMap(cp)
	}
}

func chartProbeLightFromMap(cp map[string]any) map[string]any {
	if cp == nil {
		return nil
	}
	light := map[string]any{}
	for k, v := range cp {
		if k == "bars" || k == "episodes" || k == "key_levels_series" {
			continue
		}
		light[k] = v
	}
	if _, ok := light["bar_count"]; !ok {
		if bars, ok := cp["bars"].([]any); ok {
			light["bar_count"] = len(bars)
		}
	}
	return light
}

// chartProbePayloadLight is the SSE-safe probe snapshot (no bars / episode overlays).
func chartProbePayloadLight(flow *Flow) map[string]any {
	full := chartProbePayload(flow)
	return chartProbeLightFromMap(full)
}

// PrepareSignalDiagnoseEmitCard stores rich UI data server-side and returns a light workflow_card map.
func PrepareSignalDiagnoseEmitCard(rt *runtime.Session, chat *chatsession.ChatSession, card map[string]any) map[string]any {
	if card == nil {
		return card
	}
	skill, _ := card["skill"].(string)
	if skill == "" {
		skill, _ = card["template"].(string)
	}
	if canonicalSkill(skill) != SkillSignalDiagnose {
		return card
	}
	out := map[string]any{}
	for k, v := range card {
		out[k] = v
	}
	normalizeSignalDiagnoseEmitCard(out)
	var cp map[string]any
	var eval slots.SignalEpisodeEval
	if flow := flowFromEmitCard(out, rt); flow != nil {
		if full, ev, ok := uiPayloadFromFlow(flow); ok {
			cp, eval = full, ev
		}
	}
	if cp == nil {
		if raw, ok := out["chart_probe"].(map[string]any); ok && raw["bars"] != nil {
			cp = map[string]any{}
			for k, v := range raw {
				cp[k] = v
			}
		}
	}
	if cp != nil && eval.Method == "" {
		bars, _ := cp["bars"].([]any)
		buyMerged, _ := cp["buy_merged"].([]any)
		sellMerged, _ := cp["sell_merged"].([]any)
		if len(bars) > 0 {
			opts := slots.EvaluateSignalEpisodesOptions{}
			if td, ok := out["trade_direction"].(string); ok && strings.EqualFold(td, "long_only") {
				opts.LongOnly = true
			} else if rt != nil && rt.SignalDiagnoseUIPending != nil && rt.SignalDiagnoseUIPending.LongOnly {
				opts.LongOnly = true
			}
			eval = slots.EvaluateSignalEpisodesWithOptions(bars, buyMerged, sellMerged, opts)
		}
	}
	if cp != nil {
		key := StoreSignalDiagnoseUIPayload(rt, chat, SignalDiagnoseUIEntry{
			RunID: strAny(out["run_id"]), SessionID: sessionIDFromStore(rt, chat),
			StockCode: strAny(out["stock_code"]), ChartProbe: cp, SignalEval: eval,
		})
		if key != "" {
			out["diagnose_ui_key"] = key
		}
	}
	stripHeavySignalDiagnoseCard(out)
	ensureSignalDiagnoseCompletionFields(out)
	return out
}

func strAny(v any) string {
	s, _ := v.(string)
	return strings.TrimSpace(s)
}

func normalizeSignalDiagnoseEmitCard(card map[string]any) {
	if card["skill"] == nil && card["template"] != nil {
		card["skill"] = card["template"]
	}
}

func ensureSignalDiagnoseCompletionFields(card map[string]any) {
	if strings.TrimSpace(strAny(card["status"])) == "" {
		if phase := strings.TrimSpace(strAny(card["phase"])); phase == "done" || phase == PhaseDone {
			card["status"] = StatusCompleted
		}
	}
	if strings.TrimSpace(strAny(card["card_stage"])) == "" {
		if strings.TrimSpace(strAny(card["status"])) == StatusCompleted {
			card["card_stage"] = "eval_done"
		}
	}
}

func flowFromEmitCard(card map[string]any, rt *runtime.Session) *Flow {
	flow := &Flow{Template: SkillSignalDiagnose}
	if rt != nil && rt.SignalDiagnoseUIPending != nil {
		p := rt.SignalDiagnoseUIPending
		flow.ProbeRaw = p.ProbeRaw
		flow.StockCode = p.StockCode
		if flow.StockCode == "" {
			flow.StockCode = strAny(card["stock_code"])
		}
		if len(flow.Strategies) == 0 && p.Frequency != "" {
			flow.Strategies = []StrategyItem{{Frequency: p.Frequency}}
		}
		flow.MonthsBack = p.MonthsBack
		if p.LongOnly {
			td := runtime.TradeDirectionLongOnly
			flow.TradeDirection = td
		}
	}
	if raw, ok := card["chart_probe"].(map[string]any); ok && raw["bars"] != nil {
		flow.ProbeRaw = raw
	}
	if flow.ProbeRaw == nil {
		return nil
	}
	return flow
}
