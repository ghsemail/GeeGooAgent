package chat

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

const (
	// MetaKeyLabDiagnoseEpisodeDetails persists computed lab-detail payloads per diagnose run_id.
	// FE may mirror the inner `details` map into Signal diagnose log `run.lab_episode_details`.
	MetaKeyLabDiagnoseEpisodeDetails = "lab_diagnose_episode_details"
)

type labDiagnoseEpisodeDetailsEntry struct {
	ExpiresAt time.Time                                      `json:"expires_at"`
	Details   map[string]strategydev.LabEpisodeDetail        `json:"details"`
}

func labDetailCacheTTL() time.Time {
	return time.Now().UTC().Add(labDiagnoseSnapshotTTL)
}

func loadLabEpisodeDetailsMap(meta map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	if meta == nil {
		return out
	}
	raw, ok := meta[MetaKeyLabDiagnoseEpisodeDetails]
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

func parseLabEpisodeDetailsEntry(raw json.RawMessage) (labDiagnoseEpisodeDetailsEntry, bool) {
	if len(raw) == 0 {
		return labDiagnoseEpisodeDetailsEntry{}, false
	}
	var ent labDiagnoseEpisodeDetailsEntry
	if err := json.Unmarshal(raw, &ent); err != nil {
		return labDiagnoseEpisodeDetailsEntry{}, false
	}
	if time.Now().UTC().After(ent.ExpiresAt) {
		return labDiagnoseEpisodeDetailsEntry{}, false
	}
	if ent.Details == nil {
		ent.Details = map[string]strategydev.LabEpisodeDetail{}
	}
	return ent, true
}

// CachedLabEpisodeDetail returns a persisted lab-detail payload when present and not expired.
func CachedLabEpisodeDetail(chat *chatsession.ChatSession, runID, detailRef string) (strategydev.LabEpisodeDetail, bool) {
	runID = strings.TrimSpace(runID)
	detailRef = strings.TrimSpace(detailRef)
	if chat == nil || runID == "" || detailRef == "" {
		return strategydev.LabEpisodeDetail{}, false
	}
	merged := loadLabEpisodeDetailsMap(chat.Metadata)
	raw, ok := merged[runID]
	if !ok {
		return strategydev.LabEpisodeDetail{}, false
	}
	ent, ok := parseLabEpisodeDetailsEntry(raw)
	if !ok {
		return strategydev.LabEpisodeDetail{}, false
	}
	detail, ok := ent.Details[detailRef]
	return detail, ok
}

// StoreLabEpisodeDetail saves one computed lab-detail under run_id+detail_ref (session metadata + optional runtime).
func StoreLabEpisodeDetail(chat *chatsession.ChatSession, rt *runtime.Session, runID, detailRef string, detail strategydev.LabEpisodeDetail) {
	runID = strings.TrimSpace(runID)
	detailRef = strings.TrimSpace(detailRef)
	if chat == nil || runID == "" || detailRef == "" {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := loadLabEpisodeDetailsMap(chat.Metadata)
	var ent labDiagnoseEpisodeDetailsEntry
	if raw, ok := merged[runID]; ok {
		if parsed, ok := parseLabEpisodeDetailsEntry(raw); ok {
			ent = parsed
		}
	}
	if ent.Details == nil {
		ent.Details = map[string]strategydev.LabEpisodeDetail{}
	}
	ent.ExpiresAt = labDetailCacheTTL()
	ent.Details[detailRef] = detail
	raw, err := json.Marshal(ent)
	if err != nil {
		return
	}
	merged[runID] = raw
	pruneLabDiagnoseSnapshots(merged)
	chat.Metadata[MetaKeyLabDiagnoseEpisodeDetails] = merged

	if rt != nil {
		if rt.LabDiagnoseEpisodeDetails == nil {
			rt.LabDiagnoseEpisodeDetails = map[string]json.RawMessage{}
		}
		rt.LabDiagnoseEpisodeDetails[runID] = raw
	}
}

// LabEpisodeDetailsForRun returns all cached detail_ref → detail for one diagnose run (for FE export).
func LabEpisodeDetailsForRun(chat *chatsession.ChatSession, runID string) map[string]strategydev.LabEpisodeDetail {
	runID = strings.TrimSpace(runID)
	if chat == nil || runID == "" {
		return nil
	}
	merged := loadLabEpisodeDetailsMap(chat.Metadata)
	raw, ok := merged[runID]
	if !ok {
		return nil
	}
	ent, ok := parseLabEpisodeDetailsEntry(raw)
	if !ok || len(ent.Details) == 0 {
		return nil
	}
	out := make(map[string]strategydev.LabEpisodeDetail, len(ent.Details))
	for k, v := range ent.Details {
		out[k] = v
	}
	return out
}

func syncLabEpisodeDetailsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || len(rt.LabDiagnoseEpisodeDetails) == 0 {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := loadLabEpisodeDetailsMap(chat.Metadata)
	for id, raw := range rt.LabDiagnoseEpisodeDetails {
		merged[id] = raw
	}
	pruneLabDiagnoseSnapshots(merged)
	chat.Metadata[MetaKeyLabDiagnoseEpisodeDetails] = merged
}

func loadLabEpisodeDetailsIntoSession(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	merged := loadLabEpisodeDetailsMap(chat.Metadata)
	if len(merged) == 0 {
		return
	}
	if rt.LabDiagnoseEpisodeDetails == nil {
		rt.LabDiagnoseEpisodeDetails = map[string]json.RawMessage{}
	}
	for id, raw := range merged {
		rt.LabDiagnoseEpisodeDetails[id] = raw
	}
}
