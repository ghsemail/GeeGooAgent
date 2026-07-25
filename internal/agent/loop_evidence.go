package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

var skipChatEvidenceTools = map[string]struct{}{
	"recall": {}, "clarify": {}, "read_working_state": {},
}

func (l *Loop) recordChatEvidence(ctx context.Context, session *runtime.Session, call llm.ToolCall, result tools.Result) {
	if l == nil || l.mem == nil || session == nil {
		return
	}
	if result.Status != tools.StatusOK && result.Status != tools.StatusDryRun {
		return
	}
	if _, skip := skipChatEvidenceTools[call.Name]; skip {
		return
	}
	summary := strings.TrimSpace(result.Summary)
	if len(summary) > 500 {
		summary = summary[:500]
	}
	if summary == "" {
		return
	}
	id := fmt.Sprintf("%s-%s-%d", session.ID, call.Name, time.Now().UTC().UnixNano())
	_ = l.mem.Store(ctx, memport.Record{
		Kind: memport.RecordEvidence,
		Ref: memport.EvidenceRef{
			ID:         id,
			RunID:      session.ID,
			Tool:       call.Name,
			Source:     "chat",
			Summary:    summary,
			ObservedAt: time.Now().UTC().Format(time.RFC3339),
		},
		Payload: result.Data,
	})
}
