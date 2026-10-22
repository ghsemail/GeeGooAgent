package chat

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const MetaKeySignalDiagnoseResolveCache = "signal_diagnose_resolve_cache"

type signalResolveCacheEntry struct {
	Buy           []any  `json:"buy,omitempty"`
	Sell          []any  `json:"sell,omitempty"`
	Frequency     string `json:"frequency,omitempty"`
	StrategyLabel string `json:"strategy_label,omitempty"`
	Source        string `json:"source,omitempty"`
}

func signalResolveCacheKey(flow *Flow) string {
	if flow == nil {
		return ""
	}
	payload := map[string]any{
		"strategy_query": flow.StrategyQuery,
		"catalog_type":   flow.CatalogType,
		"catalog_label":  flow.CatalogLabel,
		"buy_override":   flow.ProbeBuyOverride,
		"sell_override":  flow.ProbeSellOverride,
		"buy_merge":      flow.ProbeBuyMerge,
		"sell_merge":     flow.ProbeSellMerge,
		"frequency":      flow.ProbeFrequencyOverride,
		"months_back":    flow.MonthsBack,
	}
	if len(flow.CatalogRaw) > 0 {
		payload["catalog_signal_id"] = catalogField(flow.CatalogRaw, "signal_id", "id")
		payload["catalog_name"] = catalogField(flow.CatalogRaw, "name")
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}

func loadSignalResolveFromSession(session *runtime.Session, key string) (signalResolveCacheEntry, bool) {
	if session == nil || key == "" || session.SignalDiagnoseResolveCache == nil {
		return signalResolveCacheEntry{}, false
	}
	raw, ok := session.SignalDiagnoseResolveCache[key]
	if !ok || len(raw) == 0 {
		return signalResolveCacheEntry{}, false
	}
	var ent signalResolveCacheEntry
	if err := json.Unmarshal(raw, &ent); err != nil {
		return signalResolveCacheEntry{}, false
	}
	if len(ent.Buy) == 0 && len(ent.Sell) == 0 {
		return signalResolveCacheEntry{}, false
	}
	return ent, true
}

func syncSignalDiagnoseResolveCacheToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || len(rt.SignalDiagnoseResolveCache) == 0 {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := map[string]json.RawMessage{}
	if raw, ok := chat.Metadata[MetaKeySignalDiagnoseResolveCache]; ok {
		switch v := raw.(type) {
		case map[string]json.RawMessage:
			for k, val := range v {
				merged[k] = val
			}
		}
	}
	for k, v := range rt.SignalDiagnoseResolveCache {
		merged[k] = v
	}
	chat.Metadata[MetaKeySignalDiagnoseResolveCache] = merged
}

func loadSignalDiagnoseResolveCacheFromChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	raw, ok := chat.Metadata[MetaKeySignalDiagnoseResolveCache]
	if !ok || raw == nil {
		return
	}
	out := map[string]json.RawMessage{}
	switch v := raw.(type) {
	case map[string]json.RawMessage:
		for k, val := range v {
			out[k] = val
		}
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return
		}
		var generic map[string]json.RawMessage
		if err := json.Unmarshal(b, &generic); err != nil {
			return
		}
		out = generic
	}
	if len(out) == 0 {
		return
	}
	rt.SignalDiagnoseResolveCache = out
}

func storeSignalResolveToSession(session *runtime.Session, key string, sig slots.ResolvedSignal, source string) {
	if session == nil || key == "" || (len(sig.Buy) == 0 && len(sig.Sell) == 0) {
		return
	}
	if session.SignalDiagnoseResolveCache == nil {
		session.SignalDiagnoseResolveCache = map[string]json.RawMessage{}
	}
	ent := signalResolveCacheEntry{
		Buy: sig.Buy, Sell: sig.Sell, Frequency: sig.Frequency,
		StrategyLabel: sig.StrategyLabel, Source: strings.TrimSpace(source),
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		return
	}
	session.SignalDiagnoseResolveCache[key] = raw
}

func persistResolvedSignalCache(session *runtime.Session, key string, sig slots.ResolvedSignal, source string) {
	storeSignalResolveGlobal(key, sig, source)
	storeSignalResolveToSession(session, key, sig, source)
}

func applySignalResolveCacheEntry(flow *Flow, ent signalResolveCacheEntry) {
	if flow == nil {
		return
	}
	flow.ResolvedBuySignal = ent.Buy
	flow.ResolvedSellSignal = ent.Sell
	flow.ResolvedSignalFrequency = ent.Frequency
	flow.ResolvedStrategyLabel = ent.StrategyLabel
	flow.DiagnoseSignalSource = ent.Source
	flow.DiagnoseSignalCached = true
	flow.ProbeSignalFingerprint = fingerprintProbeSignalPlan(flow)
}
