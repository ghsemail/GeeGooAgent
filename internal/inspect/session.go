package inspect

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/lineage"
)

// SessionReport is compaction lineage for one chat session.
type SessionReport struct {
	SessionID            string
	Title                string
	MessageCount         int
	LineageRoot          string
	ParentID             string
	CompactionGeneration int
	Chain                []lineage.Record
}

// BuildSession loads lineage from a persisted chat session.
func BuildSession(chat *chatsession.ChatSession) SessionReport {
	if chat == nil {
		return SessionReport{}
	}
	parent, root, gen := chat.LineageFromMetadata()
	return SessionReport{
		SessionID: chat.ID, Title: chat.Title, MessageCount: len(chat.Messages),
		LineageRoot: root, ParentID: parent, CompactionGeneration: gen,
		Chain: chat.LineageChainFromMetadata(),
	}
}

// FormatSessionText renders session lineage.
func FormatSessionText(r SessionReport) string {
	var b strings.Builder
	b.WriteString("GeeGoo Session Lineage\n")
	b.WriteString(fmt.Sprintf("session: %s\n", r.SessionID))
	if r.Title != "" {
		b.WriteString(fmt.Sprintf("title: %s\n", r.Title))
	}
	b.WriteString(fmt.Sprintf("messages: %d  compaction_generation: %d\n", r.MessageCount, r.CompactionGeneration))
	if r.LineageRoot != "" {
		b.WriteString(fmt.Sprintf("lineage_root: %s  parent_id: %s\n", r.LineageRoot, r.ParentID))
	}
	b.WriteByte('\n')
	b.WriteString("[compaction chain]\n")
	if len(r.Chain) == 0 {
		b.WriteString("  (no compaction events yet)\n")
	} else {
		for _, rec := range r.Chain {
			b.WriteString("  " + rec.FormatLine() + "\n")
		}
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}
