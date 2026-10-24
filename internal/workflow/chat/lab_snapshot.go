package chat

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const (
	MetaKeyLabDiagnoseSnapshots = "lab_diagnose_snapshots"
	labDiagnoseSnapshotTTL      = 48 * time.Hour
	maxLabDiagnoseSnapshots     = 12
)

type labDiagnoseSnapshotEntry struct {
	ExpiresAt time.Time `json:"expires_at"`
	Flow      Flow      `json:"flow"`
}

// ArchiveLabDiagnoseFlow stores a completed lab-mode diagnose flow for lazy lab-detail.
func ArchiveLabDiagnoseFlow(session *runtime.Session, flow *Flow) {
	if session == nil || flow == nil || !flow.LabMode || flow.RunID == "" {
		return
	}
	if session.LabDiagnoseSnapshots == nil {
		session.LabDiagnoseSnapshots = map[string]json.RawMessage{}
	}
	ent := labDiagnoseSnapshotEntry{
		ExpiresAt: time.Now().UTC().Add(labDiagnoseSnapshotTTL),
		Flow:      *flow,
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		return
	}
	session.LabDiagnoseSnapshots[flow.RunID] = raw
	pruneLabDiagnoseSnapshots(session.LabDiagnoseSnapshots)
}

func pruneLabDiagnoseSnapshots(m map[string]json.RawMessage) {
	if len(m) <= maxLabDiagnoseSnapshots {
		return
	}
	now := time.Now().UTC()
	for id, raw := range m {
		var ent labDiagnoseSnapshotEntry
		if err := json.Unmarshal(raw, &ent); err != nil || !ent.ExpiresAt.After(now) {
			delete(m, id)
		}
	}
	for len(m) > maxLabDiagnoseSnapshots {
		for id := range m {
			delete(m, id)
			break
		}
	}
}

func syncLabDiagnoseSnapshotsToChat(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || len(rt.LabDiagnoseSnapshots) == 0 {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := loadLabSnapshotMap(chat.Metadata)
	for id, raw := range rt.LabDiagnoseSnapshots {
		merged[id] = raw
	}
	pruneLabDiagnoseSnapshots(merged)
	chat.Metadata[MetaKeyLabDiagnoseSnapshots] = merged
}

func loadLabSnapshotMap(meta map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	raw, ok := meta[MetaKeyLabDiagnoseSnapshots]
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

func loadLabDiagnoseSnapshotsIntoSession(chat *chatsession.ChatSession, rt *runtime.Session) {
	if chat == nil || rt == nil || chat.Metadata == nil {
		return
	}
	merged := loadLabSnapshotMap(chat.Metadata)
	if len(merged) == 0 {
		return
	}
	if rt.LabDiagnoseSnapshots == nil {
		rt.LabDiagnoseSnapshots = map[string]json.RawMessage{}
	}
	for id, raw := range merged {
		rt.LabDiagnoseSnapshots[id] = raw
	}
}

// UpsertLabDiagnoseSnapshot writes or refreshes one archived lab flow (e.g. after lazy MTF cache).
func UpsertLabDiagnoseSnapshot(chat *chatsession.ChatSession, flow *Flow) {
	if chat == nil || flow == nil || !flow.LabMode || flow.RunID == "" {
		return
	}
	if chat.Metadata == nil {
		chat.Metadata = map[string]any{}
	}
	merged := loadLabSnapshotMap(chat.Metadata)
	ent := labDiagnoseSnapshotEntry{
		ExpiresAt: time.Now().UTC().Add(labDiagnoseSnapshotTTL),
		Flow:      *flow,
	}
	raw, err := json.Marshal(ent)
	if err != nil {
		return
	}
	merged[flow.RunID] = raw
	pruneLabDiagnoseSnapshots(merged)
	chat.Metadata[MetaKeyLabDiagnoseSnapshots] = merged
}

// MustLabDiagnoseSnapshotJSON marshals a non-expired lab snapshot entry (tests and fixtures).
func MustLabDiagnoseSnapshotJSON(flow Flow) json.RawMessage {
	ent := labDiagnoseSnapshotEntry{
		ExpiresAt: time.Now().UTC().Add(labDiagnoseSnapshotTTL),
		Flow:      flow,
	}
	raw, _ := json.Marshal(ent)
	return raw
}

// ResolveFlowForLabDetail returns active or archived flow for lab-detail.
func ResolveFlowForLabDetail(chat *chatsession.ChatSession, runID string) *Flow {
	runID = strings.TrimSpace(runID)
	if active := FlowFromChat(chat); active != nil {
		if runID == "" || runID == active.RunID {
			return active
		}
	}
	if chat == nil || chat.Metadata == nil || runID == "" {
		return nil
	}
	merged := loadLabSnapshotMap(chat.Metadata)
	raw, ok := merged[runID]
	if !ok || len(raw) == 0 {
		return nil
	}
	var ent labDiagnoseSnapshotEntry
	if err := json.Unmarshal(raw, &ent); err != nil {
		return nil
	}
	if time.Now().UTC().After(ent.ExpiresAt) {
		return nil
	}
	if ent.Flow.RunID == "" {
		return nil
	}
	flow := ent.Flow
	normalizeOnLoad(&flow)
	return &flow
}
