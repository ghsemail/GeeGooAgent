package strategydev

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/priceanalysis"
)

type labBarsCacheKey struct {
	session string
	run     string
	code    string
	freq    string
	months  int
}

type labBarsCacheEntry struct {
	bars      []any
	ok        bool
	expiresAt time.Time
}

var labBarsFetchCache sync.Map

const labBarsFetchCacheTTL = 48 * time.Hour

func labBarsCacheKeyFrom(sessionID, runID, code, freq string, months int) labBarsCacheKey {
	return labBarsCacheKey{
		session: strings.TrimSpace(sessionID),
		run:     strings.TrimSpace(runID),
		code:    strings.TrimSpace(code),
		freq:    strings.TrimSpace(freq),
		months:  months,
	}
}

// CachedLabBarsFetcher wraps a BarsFetcher with per-diagnose-run reuse.
// Lab-api jobs are keyed by code/freq/months_back; episode anchors truncate locally in priceanalysis.Run.
// asOf is never forwarded to lab-api on cache miss (full window once, callers filter if needed).
type CachedLabBarsFetcher struct {
	SessionID string
	RunID     string
	Inner     priceanalysis.BarsFetcher
}

func (c CachedLabBarsFetcher) FetchBars(ctx context.Context, code, freq string, months int, asOf *time.Time) ([]any, bool) {
	if c.Inner == nil {
		return nil, false
	}
	key := labBarsCacheKeyFrom(c.SessionID, c.RunID, code, freq, months)
	if key.run == "" {
		return c.Inner.FetchBars(ctx, code, freq, months, nil)
	}
	if raw, ok := labBarsFetchCache.Load(key); ok {
		if ent, ok := raw.(labBarsCacheEntry); ok && time.Now().UTC().Before(ent.expiresAt) {
			return cloneLabBarsSlice(ent.bars), ent.ok
		}
		labBarsFetchCache.Delete(key)
	}
	bars, ok := c.Inner.FetchBars(ctx, code, freq, months, nil)
	// Do not negative-cache failures: anchor daily and PA daily often share months_back;
	// a phase-timeout miss must not poison the whole diagnose run.
	if ok && len(bars) > 0 {
		labBarsFetchCache.Store(key, labBarsCacheEntry{
			bars:      cloneLabBarsSlice(bars),
			ok:        true,
			expiresAt: time.Now().UTC().Add(labBarsFetchCacheTTL),
		})
	}
	return bars, ok
}

func cloneLabBarsSlice(in []any) []any {
	if len(in) == 0 {
		return nil
	}
	out := make([]any, len(in))
	copy(out, in)
	return out
}

// WrapCachedLabBarsFetcher returns a run-scoped caching fetcher when runID is set.
func WrapCachedLabBarsFetcher(sessionID, runID string, inner priceanalysis.BarsFetcher) priceanalysis.BarsFetcher {
	if inner == nil || strings.TrimSpace(runID) == "" {
		return inner
	}
	return CachedLabBarsFetcher{SessionID: sessionID, RunID: runID, Inner: inner}
}
