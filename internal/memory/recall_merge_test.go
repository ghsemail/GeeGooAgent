package memory

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

func TestMergeRecallHitsHybrid(t *testing.T) {
	fts := []chatsession.SessionRecallHit{
		{SessionID: "a", Score: 5, Snippet: "fts a"},
		{SessionID: "b", Score: 3, Snippet: "fts b"},
	}
	vec := []chatsession.SessionRecallHit{
		{SessionID: "b", Score: 2, Snippet: "vec b longer"},
		{SessionID: "c", Score: 1, Snippet: "vec c"},
	}
	merged, source := mergeRecallHits(fts, vec, 5)
	if source != "hybrid" {
		t.Fatalf("source=%q want hybrid", source)
	}
	if len(merged) != 3 {
		t.Fatalf("len=%d want 3", len(merged))
	}
	ids := map[string]bool{}
	for _, h := range merged {
		ids[h.SessionID] = true
	}
	for _, want := range []string{"a", "b", "c"} {
		if !ids[want] {
			t.Fatalf("missing session %s", want)
		}
	}
}

func TestMergeRecallHitsVectorOnly(t *testing.T) {
	merged, source := mergeRecallHits(nil, []chatsession.SessionRecallHit{
		{SessionID: "x", Score: 1, Snippet: "only vector"},
	}, 3)
	if source != "vector" || len(merged) != 1 {
		t.Fatalf("source=%q merged=%+v", source, merged)
	}
}
