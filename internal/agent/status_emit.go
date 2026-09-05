package agent

import (
	"fmt"
	"sync"
	"time"
)

// emitStatus sends a lightweight human-readable progress line for chat SSE / harness UI.
func (l *Loop) emitStatus(phase, message string) {
	if message == "" {
		return
	}
	payload := map[string]any{"phase": phase, "message": message}
	if phase != "" {
		payload["phase"] = phase
	}
	l.emit("status", payload)
}

// startStatusHeartbeat emits elapsed wait lines while a blocking model call runs.
func (l *Loop) startStatusHeartbeat(phase, prefix string, every time.Duration) func() {
	if l == nil || every <= 0 {
		return func() {}
	}
	done := make(chan struct{})
	var once sync.Once
	started := time.Now()
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
				sec := int(time.Since(started).Seconds())
				if sec < 1 {
					sec = 1
				}
				l.emitStatus(phase, fmt.Sprintf("%s（已等待 %ds，等辅助模型返回）", prefix, sec))
			}
		}
	}()
	return func() {
		once.Do(func() { close(done) })
	}
}
