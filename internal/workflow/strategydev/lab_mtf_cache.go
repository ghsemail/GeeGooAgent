package strategydev

import (
	"strings"
	"sync"
	"time"
)

type labMTFCacheEntry struct {
	bundle    *KlinePatternMTFBundle
	expiresAt time.Time
}

var labMTFCache sync.Map

const labMTFCacheTTL = 48 * time.Hour

func labMTFCacheKey(sessionID, runID string) string {
	return strings.TrimSpace(sessionID) + "\x00" + strings.TrimSpace(runID)
}

// CachedLabKlineMTF returns a cached MTF bundle for one diagnose run (same session).
func CachedLabKlineMTF(sessionID, runID string) *KlinePatternMTFBundle {
	key := labMTFCacheKey(sessionID, runID)
	raw, ok := labMTFCache.Load(key)
	if !ok {
		return nil
	}
	ent, ok := raw.(labMTFCacheEntry)
	if !ok || ent.bundle == nil || time.Now().UTC().After(ent.expiresAt) {
		labMTFCache.Delete(key)
		return nil
	}
	return ent.bundle
}

// StoreCachedLabKlineMTF saves MTF probes for subsequent lab-detail calls.
func StoreCachedLabKlineMTF(sessionID, runID string, bundle *KlinePatternMTFBundle) {
	if bundle == nil || strings.TrimSpace(runID) == "" {
		return
	}
	key := labMTFCacheKey(sessionID, runID)
	labMTFCache.Store(key, labMTFCacheEntry{
		bundle:    bundle,
		expiresAt: time.Now().UTC().Add(labMTFCacheTTL),
	})
}
