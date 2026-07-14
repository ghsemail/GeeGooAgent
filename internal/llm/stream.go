package llm

import "context"

// StreamDelta is one incremental piece of a streaming completion.
type StreamDelta struct {
	Content          string
	ReasoningContent string
}

// StreamHandler receives content as the model generates it.
type StreamHandler func(delta StreamDelta)

type streamCtxKey struct{}

// WithStreamHandler attaches a stream callback to ctx (safe for concurrent requests).
func WithStreamHandler(ctx context.Context, h StreamHandler) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if h == nil {
		return ctx
	}
	return context.WithValue(ctx, streamCtxKey{}, h)
}

// StreamHandlerFrom returns the callback attached by WithStreamHandler, or nil.
func StreamHandlerFrom(ctx context.Context) StreamHandler {
	if ctx == nil {
		return nil
	}
	h, _ := ctx.Value(streamCtxKey{}).(StreamHandler)
	return h
}

// Streamer is an optional Provider capability for SSE / token streaming.
type Streamer interface {
	ChatStream(
		ctx context.Context,
		messages []Message,
		tools []ToolSchema,
		temperature float64,
		maxTokens int,
		onDelta StreamHandler,
	) (*Response, error)
}
