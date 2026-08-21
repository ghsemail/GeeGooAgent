package step

import (
	"context"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
)

// Step is one workflow step.
type Step struct {
	Name           string
	Tool           string
	Arguments      map[string]any
	ArgFunc        func(*memory.PreMarketWorking) map[string]any
	ContextArgFunc func(context.Context, *memory.PreMarketWorking) (map[string]any, error)
}

// Args resolves step arguments.
func (s Step) Args(ctx context.Context, working *memory.PreMarketWorking) (map[string]any, error) {
	if s.ContextArgFunc != nil {
		return s.ContextArgFunc(ctx, working)
	}
	if s.ArgFunc != nil {
		return s.ArgFunc(working), nil
	}
	if s.Arguments == nil {
		return map[string]any{}, nil
	}
	out := map[string]any{}
	for k, v := range s.Arguments {
		out[k] = v
	}
	return out, nil
}
