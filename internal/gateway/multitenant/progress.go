package multitenant

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/gateway"
)

type progressPublisher struct {
	ctx     context.Context
	ed      gateway.EditableMessenger
	chatID  string
	replyTo string

	mu        sync.Mutex
	lines     []string
	msgID     string
	lastFlush time.Time
	dirty     bool
}

func newProgressPublisher(ctx context.Context, ed gateway.EditableMessenger, chatID, replyTo string) *progressPublisher {
	return &progressPublisher{ctx: ctx, ed: ed, chatID: chatID, replyTo: replyTo}
}

func (p *progressPublisher) OnEvent(event string, data map[string]any) {
	line, ok := gateway.FormatProgressLine(event, data)
	if !ok || p == nil || p.ed == nil {
		return
	}
	force := event == "tool_done" || event == "tool_intercepted" || event == "error"
	p.mu.Lock()
	p.lines = append(p.lines, line)
	if len(p.lines) > 40 {
		p.lines = p.lines[len(p.lines)-40:]
	}
	p.dirty = true
	shouldFlush := force || p.msgID == "" || time.Since(p.lastFlush) >= 800*time.Millisecond
	p.mu.Unlock()
	if shouldFlush {
		p.Flush()
	}
}

func (p *progressPublisher) Flush() {
	if p == nil || p.ed == nil {
		return
	}
	p.mu.Lock()
	if !p.dirty && p.msgID != "" {
		p.mu.Unlock()
		return
	}
	body := gateway.RenderProgressMarkdown(append([]string(nil), p.lines...))
	msgID := p.msgID
	chatID := p.chatID
	replyTo := p.replyTo
	p.dirty = false
	p.lastFlush = time.Now()
	p.mu.Unlock()

	ctx := p.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if msgID == "" {
		id, err := p.ed.SendTextID(ctx, gateway.OutboundText{ChatID: chatID, Text: body, ReplyToID: replyTo})
		if err != nil {
			slog.Warn("gateway: progress send failed", "err", err)
			p.mu.Lock()
			p.dirty = true
			p.mu.Unlock()
			return
		}
		p.mu.Lock()
		p.msgID = id
		p.mu.Unlock()
		return
	}
	if err := p.ed.EditText(ctx, msgID, body); err != nil {
		slog.Warn("gateway: progress edit failed", "err", err, "message_id", msgID)
		p.mu.Lock()
		p.dirty = true
		p.mu.Unlock()
	}
}
