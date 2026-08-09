package gateway

import "context"

// InboundHandler receives normalized events from adapters.
type InboundHandler func(ctx context.Context, ev InboundEvent) error

// PlatformAdapter is one IM backend (Feishu, etc.).
type PlatformAdapter interface {
	Platform() Platform
	// Connect starts receiving events until ctx is cancelled or Disconnect.
	Connect(ctx context.Context, onInbound InboundHandler) error
	Disconnect(ctx context.Context) error
	SendText(ctx context.Context, msg OutboundText) error
	// SendMedia is reserved for M2; may return ErrNotImplemented.
	SendMedia(ctx context.Context, chatID string, filename string, data []byte, mime string) error
	Configured() bool
	Status() AdapterStatus
}

// ProcessingIndicator is optional: show in-progress UX (e.g. Feishu Typing reaction).
type ProcessingIndicator interface {
	MarkProcessing(ctx context.Context, messageID string) error
	ClearProcessing(ctx context.Context, messageID string) error
	MarkFailed(ctx context.Context, messageID string) error
}

// AdapterStatus is a snapshot for `geegoo gateway status`.
type AdapterStatus struct {
	Platform  Platform
	Configured bool
	Connected bool
	Detail    string
}

// ErrNotImplemented marks stubbed M2+ methods.
type ErrNotImplemented struct {
	Feature string
}

func (e ErrNotImplemented) Error() string {
	return "gateway: not implemented: " + e.Feature
}
