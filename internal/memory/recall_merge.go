package memory

import (
	"sort"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
)

// mergeRecallHits combines FTS and vector session hits, deduping by session id.
func mergeRecallHits(fts, vector []chatsession.SessionRecallHit, limit int) ([]chatsession.SessionRecallHit, string) {
	if limit <= 0 {
		limit = 5
	}
	type merged struct {
		hit       chatsession.SessionRecallHit
		fromFTS   bool
		fromVector bool
	}
	byID := map[string]*merged{}
	order := []string{}

	add := func(hit chatsession.SessionRecallHit, ftsHit, vectorHit bool) {
		if hit.SessionID == "" {
			return
		}
		ex, ok := byID[hit.SessionID]
		if !ok {
			byID[hit.SessionID] = &merged{hit: hit, fromFTS: ftsHit, fromVector: vectorHit}
			order = append(order, hit.SessionID)
			return
		}
		if ftsHit {
			ex.fromFTS = true
			if hit.Score > ex.hit.Score {
				ex.hit.Score = hit.Score
			}
		}
		if vectorHit {
			ex.fromVector = true
			// Boost hybrid matches; vector-only uses rank score from semanticChunksToSessionHits.
			ex.hit.Score += hit.Score + 2
		}
		if len(hit.Snippet) > len(ex.hit.Snippet) {
			ex.hit.Snippet = hit.Snippet
		}
		if hit.UpdatedAt > ex.hit.UpdatedAt {
			ex.hit.UpdatedAt = hit.UpdatedAt
		}
	}

	for _, h := range fts {
		add(h, true, false)
	}
	for _, h := range vector {
		add(h, false, true)
	}

	out := make([]chatsession.SessionRecallHit, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id].hit)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].UpdatedAt > out[j].UpdatedAt
	})
	if len(out) > limit {
		out = out[:limit]
	}

	ftsN, vecN := 0, 0
	for _, id := range order {
		m := byID[id]
		if m.fromFTS {
			ftsN++
		}
		if m.fromVector {
			vecN++
		}
	}
	source := "none"
	switch {
	case ftsN > 0 && vecN > 0:
		source = "hybrid"
	case ftsN > 0:
		source = "fts"
	case vecN > 0:
		source = "vector"
	}
	return out, source
}
