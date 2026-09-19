package keylevelctx

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/stockfmt"
)

// Fetcher loads engine key levels for a symbol (typically GeeGooSignal /getSupportingPrice).
type Fetcher interface {
	FetchKeyLevels(ctx context.Context, code string) (stockfmt.KeyLevels, error)
}

type fetcherKey struct{}

// ContextWithFetcher attaches a key level fetcher to ctx.
func ContextWithFetcher(ctx context.Context, f Fetcher) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if f == nil {
		return ctx
	}
	return context.WithValue(ctx, fetcherKey{}, f)
}

// From returns the fetcher attached to ctx, if any.
func From(ctx context.Context) Fetcher {
	if ctx == nil {
		return nil
	}
	f, _ := ctx.Value(fetcherKey{}).(Fetcher)
	return f
}
