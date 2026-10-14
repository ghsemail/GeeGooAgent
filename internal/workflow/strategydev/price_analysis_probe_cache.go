package strategydev

import (
	"context"
	"sync"
	"time"
)

type priceProbeCacheKey struct {
	code   string
	freq   string
	months int
	asOf   string
}

type priceProbeCacheEntry struct {
	bars []any
	ok   bool
}

type priceProbeCache struct {
	mu sync.Mutex
	m  map[priceProbeCacheKey]priceProbeCacheEntry
}

type priceProbeCacheCtxKey struct{}

// ContextWithPriceProbeCache attaches a per-request probe cache for multi-TF price analysis fetches.
func ContextWithPriceProbeCache(ctx context.Context) context.Context {
	if priceProbeCacheFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, priceProbeCacheCtxKey{}, &priceProbeCache{
		m: make(map[priceProbeCacheKey]priceProbeCacheEntry),
	})
}

func priceProbeCacheFromContext(ctx context.Context) *priceProbeCache {
	if ctx == nil {
		return nil
	}
	c, _ := ctx.Value(priceProbeCacheCtxKey{}).(*priceProbeCache)
	return c
}

func probeCacheKey(code, freq string, months int, asOf *time.Time) priceProbeCacheKey {
	k := priceProbeCacheKey{code: code, freq: freq, months: months}
	if asOf != nil && !asOf.IsZero() {
		k.asOf = formatPriceAnalysisAsOf(*asOf)
	}
	return k
}

func lookupPriceProbeCache(ctx context.Context, key priceProbeCacheKey) (bars []any, ok bool, hit bool) {
	cache := priceProbeCacheFromContext(ctx)
	if cache == nil {
		return nil, false, false
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	ent, found := cache.m[key]
	if !found {
		return nil, false, false
	}
	return ent.bars, ent.ok, true
}

func storePriceProbeCache(ctx context.Context, key priceProbeCacheKey, bars []any, ok bool) {
	cache := priceProbeCacheFromContext(ctx)
	if cache == nil {
		return
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.m[key] = priceProbeCacheEntry{bars: bars, ok: ok}
}
