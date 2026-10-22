package chat

import (
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

const signalResolveGlobalCacheTTL = 30 * time.Minute

type signalResolveGlobalEntry struct {
	ent       signalResolveCacheEntry
	expiresAt time.Time
}

var signalResolveGlobalCache sync.Map

func loadSignalResolveGlobal(key string) (signalResolveCacheEntry, bool) {
	if key == "" {
		return signalResolveCacheEntry{}, false
	}
	raw, ok := signalResolveGlobalCache.Load(key)
	if !ok {
		return signalResolveCacheEntry{}, false
	}
	ent, ok := raw.(signalResolveGlobalEntry)
	if !ok || time.Now().UTC().After(ent.expiresAt) {
		signalResolveGlobalCache.Delete(key)
		return signalResolveCacheEntry{}, false
	}
	if len(ent.ent.Buy) == 0 && len(ent.ent.Sell) == 0 {
		return signalResolveCacheEntry{}, false
	}
	return ent.ent, true
}

func storeSignalResolveGlobal(key string, sig slots.ResolvedSignal, source string) {
	if key == "" || (len(sig.Buy) == 0 && len(sig.Sell) == 0) {
		return
	}
	ent := signalResolveCacheEntry{
		Buy: sig.Buy, Sell: sig.Sell, Frequency: sig.Frequency,
		StrategyLabel: sig.StrategyLabel, Source: source,
	}
	signalResolveGlobalCache.Store(key, signalResolveGlobalEntry{
		ent:       ent,
		expiresAt: time.Now().UTC().Add(signalResolveGlobalCacheTTL),
	})
}
