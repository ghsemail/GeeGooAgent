package chat

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func TestResolveFlowForLabDetailFromArchive(t *testing.T) {
	flow := &Flow{
		RunID: "flow-20261022T120304", Template: SkillSignalDiagnose, LabMode: true,
		ProbeBuyHits: 12, ProbeSellHits: 12,
		ProbeRaw: map[string]any{"bars": []any{map[string]any{"close": 1.0}}},
	}
	session := runtime.NewSession()
	ArchiveLabDiagnoseFlow(session, flow)
	chat := &chatsession.ChatSession{ID: "s1", Metadata: map[string]any{}}
	rt := runtime.NewSession()
	syncLabDiagnoseSnapshotsToChat(chat, session)

	got := ResolveFlowForLabDetail(chat, flow.RunID)
	if got == nil {
		t.Fatal("expected archived flow")
	}
	if got.ProbeBuyHits != 12 || got.ProbeSellHits != 12 {
		t.Fatalf("hits=%d/%d", got.ProbeBuyHits, got.ProbeSellHits)
	}
	_ = rt
}

func TestLabSnapshotExpires(t *testing.T) {
	ent := labDiagnoseSnapshotEntry{
		ExpiresAt: time.Now().UTC().Add(-time.Hour),
		Flow:      Flow{RunID: "flow-old", LabMode: true},
	}
	raw, _ := json.Marshal(ent)
	chat := &chatsession.ChatSession{
		Metadata: map[string]any{
			MetaKeyLabDiagnoseSnapshots: map[string]json.RawMessage{"flow-old": raw},
		},
	}
	if ResolveFlowForLabDetail(chat, "flow-old") != nil {
		t.Fatal("expected expired snapshot removed")
	}
}

func TestNewRunIDFormat(t *testing.T) {
	id := newRunID()
	if id == "" || id == "flow-20261022T%H%M%S" {
		t.Fatalf("bad run id: %q", id)
	}
	if len(id) < len("flow-20261022T120304") {
		t.Fatalf("run id too short: %q", id)
	}
}
