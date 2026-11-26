package llm

// WithSingleFallback returns a copy of the gateway that tries the primary model and
// at most one fallback, each once. The original gateway is not modified.
func (g *Gateway) WithSingleFallback() *Gateway {
	if g == nil {
		return nil
	}
	out := *g
	if len(out.fallbacks) > 1 {
		out.fallbacks = out.fallbacks[:1:1]
	}
	out.config.MaxRetries = 1
	return &out
}
