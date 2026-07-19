package agent

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/lineage"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func (l *Loop) applyCompression(ctx context.Context, session *runtime.Session, messages []llm.Message) []llm.Message {
	return l.runCompression(ctx, session, messages, false)
}

func (l *Loop) applyHygiene(ctx context.Context, session *runtime.Session, messages []llm.Message) []llm.Message {
	return l.runCompression(ctx, session, messages, true)
}

func (l *Loop) runCompression(ctx context.Context, session *runtime.Session, messages []llm.Message, hygiene bool) []llm.Message {
	if l.compressor == nil {
		return messages
	}
	est := session.LastPromptTokens
	if est <= 0 {
		est = prompt.EstimateTokens(session.Messages)
	}
	var (
		out        []llm.Message
		did        bool
		newSummary string
		err        error
	)
	if hygiene {
		if !l.compressor.ShouldHygiene(est, len(session.Messages)) {
			return messages
		}
		out, did, newSummary, err = l.compressor.CompressHygiene(ctx, session.Messages, session.PreviousSummary, est)
	} else {
		if !l.compressor.ShouldCompress(est, len(session.Messages)) {
			return messages
		}
		out, did, newSummary, err = l.compressor.Compress(ctx, session.Messages, session.PreviousSummary, est)
	}
	if err != nil || !did {
		return messages
	}
	before := len(session.Messages)
	tokensAfter := prompt.EstimateTokens(out)
	session.Messages = out
	session.PreviousSummary = newSummary
	session.LastPromptTokens = tokensAfter
	recordCompactionLineage(session, before, len(out), est, tokensAfter, hygiene, newSummary)
	advanceLineage(session)
	event := "context_compressed"
	if hygiene {
		event = "context_hygiene"
	}
	l.emit(event, map[string]any{
		"before_msgs":             before,
		"after_msgs":              len(out),
		"estimated_tokens_before": est,
		"summary_chars":           len(session.PreviousSummary),
		"hygiene":                 hygiene,
		"parent_id":               session.ParentID,
		"lineage_root":            session.LineageRoot,
		"compaction_generation":   session.CompactionGeneration,
	})
	return session.LLMMessages()
}

func recordCompactionLineage(session *runtime.Session, msgsBefore, msgsAfter, tokensBefore, tokensAfter int, hygiene bool, summary string) {
	if session == nil {
		return
	}
	gen := session.CompactionGeneration + 1
	parent := session.LineageRoot
	if parent == "" {
		parent = session.ID
	}
	if session.CompactionGeneration > 0 {
		parent = fmt.Sprintf("%s#g%d", session.LineageRoot, session.CompactionGeneration)
		if session.LineageRoot == "" {
			parent = fmt.Sprintf("%s#g%d", session.ID, session.CompactionGeneration)
		}
	}
	session.LineageChain = append(session.LineageChain, lineage.Record{
		Generation: gen, ParentID: parent, Hygiene: hygiene,
		MsgsBefore: msgsBefore, MsgsAfter: msgsAfter,
		TokensBefore: tokensBefore, TokensAfter: tokensAfter,
		SummaryHash: lineage.SummaryHashPrefix(summary), SummaryChars: len(summary),
	})
}

// advanceLineage records a Hermes-style child generation after compaction.
func advanceLineage(session *runtime.Session) {
	if session == nil {
		return
	}
	if session.LineageRoot == "" {
		session.LineageRoot = session.ID
	}
	if session.CompactionGeneration <= 0 {
		session.ParentID = session.LineageRoot
	} else {
		session.ParentID = fmt.Sprintf("%s#g%d", session.LineageRoot, session.CompactionGeneration)
	}
	session.CompactionGeneration++
}
