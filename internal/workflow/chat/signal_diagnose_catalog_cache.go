package chat

import (
	"context"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const signalCatalogToolCacheTTL = 10 * time.Minute

type catalogToolCache struct {
	mu sync.RWMutex
	m  map[string]catalogToolCacheEntry
}

type catalogToolCacheEntry struct {
	res       tools.Result
	expiresAt time.Time
}

func (c *catalogToolCache) get(name string) (tools.Result, bool) {
	if c == nil {
		return tools.Result{}, false
	}
	c.mu.RLock()
	ent, ok := c.m[name]
	c.mu.RUnlock()
	if !ok || time.Now().UTC().After(ent.expiresAt) {
		return tools.Result{}, false
	}
	return ent.res, true
}

func (c *catalogToolCache) put(name string, res tools.Result) {
	if c == nil || res.Status != tools.StatusOK {
		return
	}
	c.mu.Lock()
	if c.m == nil {
		c.m = map[string]catalogToolCacheEntry{}
	}
	c.m[name] = catalogToolCacheEntry{res: res, expiresAt: time.Now().UTC().Add(signalCatalogToolCacheTTL)}
	c.mu.Unlock()
}

var signalDiagnoseCatalogCache catalogToolCache

func cachedCatalogToolRunner(inner ToolRunner) ToolRunner {
	if inner == nil {
		return inner
	}
	return func(ctx context.Context, req tools.CallRequest, toolCtx tools.Context) tools.Result {
		switch req.Name {
		case "get_signal_combinations", "get_index_signals":
			if res, ok := signalDiagnoseCatalogCache.get(req.Name); ok {
				return res
			}
			res := inner(ctx, req, toolCtx)
			signalDiagnoseCatalogCache.put(req.Name, res)
			return res
		default:
			return inner(ctx, req, toolCtx)
		}
	}
}
