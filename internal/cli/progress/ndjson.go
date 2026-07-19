package progress

import (
	"io"
	"sync"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// NDJSONSink writes agent loop events as newline-delimited JSON.
type NDJSONSink struct {
	w  io.Writer
	mu sync.Mutex
}

// NewNDJSONSink creates a sink targeting w.
func NewNDJSONSink(w io.Writer) *NDJSONSink {
	return &NDJSONSink{w: w}
}

// EmitProgress implements Sink.
func (s *NDJSONSink) EmitProgress(event string, data map[string]any) {
	if s == nil || s.w == nil {
		return
	}
	line, err := runtime.ProgressToAgentEvent(event, data).EncodeLine()
	if err != nil {
		return
	}
	s.mu.Lock()
	_, _ = s.w.Write(line)
	s.mu.Unlock()
}
