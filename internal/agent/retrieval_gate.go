package agent

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
)

// RetrievalGateDecision is the outcome of the lightweight pre-turn memory gate.
type RetrievalGateDecision struct {
	Decision string // skip | retrieve
	Reason   string
	Hits     int
}

var retrievalSkipPrefixes = []string{
	"你好", "您好", "hi", "hello", "hey", "谢谢", "感谢", "好的", "ok", "okay",
	"嗯", "哦", "在吗", "在不在", "/",
}

var retrievalTriggerSubstrings = []string{
	"之前", "上次", "以前", "记得", "还记得", "历史", "查过", "看过", "说过", "聊过",
	"我们讨论", "继续", "接着", "刚才", "早期", "回忆", "过往", "昨天", "前天",
	"recall", "remember",
}

// DecideRetrievalGate applies fast heuristics (no LLM) for skip vs session FTS recall.
func DecideRetrievalGate(userText string) RetrievalGateDecision {
	text := strings.TrimSpace(userText)
	if text == "" {
		return RetrievalGateDecision{Decision: "skip", Reason: "empty message"}
	}
	lower := strings.ToLower(text)
	if strings.HasPrefix(lower, "/") {
		return RetrievalGateDecision{Decision: "skip", Reason: "slash command"}
	}
	for _, p := range retrievalSkipPrefixes {
		if strings.HasPrefix(lower, strings.ToLower(p)) && utf8.RuneCountInString(text) <= 12 {
			return RetrievalGateDecision{Decision: "skip", Reason: "short greeting or ack"}
		}
	}
	if utf8.RuneCountInString(text) <= 3 && !containsStockHint(text) {
		return RetrievalGateDecision{Decision: "skip", Reason: "too short for memory lookup"}
	}
	for _, kw := range retrievalTriggerSubstrings {
		if strings.Contains(lower, kw) {
			return RetrievalGateDecision{
				Decision: "retrieve",
				Reason:   "history or continuation cue: " + kw,
			}
		}
	}
	if containsStockHint(text) {
		return RetrievalGateDecision{
			Decision: "retrieve",
			Reason:   "stock or market query — check past session context",
		}
	}
	return RetrievalGateDecision{Decision: "skip", Reason: "no memory retrieval needed"}
}

func containsStockHint(text string) bool {
	lower := strings.ToLower(text)
	for _, kw := range []string{
		"股", "行情", "涨跌", "股价", "市值", "bot", "macd", "k线",
		"price", "ticker", "a股", "港股", "美股",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	if strings.Contains(text, ".HK") || strings.Contains(text, ".SZ") || strings.Contains(text, ".SS") {
		return true
	}
	for _, tok := range strings.Fields(text) {
		if len(tok) == 6 && tok[0] >= '0' && tok[0] <= '9' {
			return true
		}
	}
	return false
}

func formatGateMemory(hits []memport.RecallHit) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	for i, h := range hits {
		if i >= 3 {
			break
		}
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "- session %s (score %d): %s", h.ID, h.Score, strings.TrimSpace(h.Snippet))
	}
	return b.String()
}

func (l *Loop) injectGateMemory(session *runtime.Session, block, source string) {
	if session == nil || strings.TrimSpace(block) == "" {
		return
	}
	label := "local FTS"
	switch source {
	case "vector":
		label = "semantic vector search"
	case "hybrid":
		label = "FTS + semantic vector (hybrid)"
	case "fts":
		label = "local FTS"
	}
	mem := llm.Message{
		Role: llm.RoleSystem,
		Content: fmt.Sprintf("Retrieved from past chat sessions (%s). Use only if relevant:\n%s", label, block),
	}
	n := len(session.Messages)
	if n >= 1 && session.Messages[n-1].Role == llm.RoleUser {
		session.Messages = append(session.Messages[:n-1], append([]llm.Message{mem}, session.Messages[n-1:]...)...)
		return
	}
	session.AppendMessage(mem)
}

func (l *Loop) runRetrievalGate(ctx context.Context, session *runtime.Session, userText string) {
	l.emitStatus("gate", "检查是否需要检索历史记忆…")
	decision := DecideRetrievalGate(userText)
	if decision.Decision == "retrieve" && l.mem != nil {
		res, err := l.mem.Recall(ctx, memport.RecallQuery{
			Kind:             memport.RecallSession,
			Query:            userText,
			ExcludeSessionID: session.ID,
			UserID:           session.UserID,
			Limit:            3,
			ScanLimit:        20,
		})
		if err != nil {
			decision.Decision = "skip"
			decision.Reason = "recall error: " + err.Error()
		} else if len(res.Hits) == 0 {
			decision.Decision = "skip"
			decision.Reason = "no matching past sessions"
		} else {
			source := "fts"
			if res.Data != nil {
				if v, ok := res.Data["recall_source"].(string); ok && v != "" {
					source = v
				}
			}
			l.emitStatus("gate", fmt.Sprintf("检索到 %d 条相关历史会话（%s）", len(res.Hits), source))
			if block := formatGateMemory(res.Hits); block != "" {
				l.injectGateMemory(session, block, source)
			}
			decision.Hits = len(res.Hits)
			if decision.Reason == "" {
				decision.Reason = fmt.Sprintf("matched %d past session(s) via %s", len(res.Hits), source)
			}
			l.emit("gate", map[string]any{
				"decision":       decision.Decision,
				"reason":         decision.Reason,
				"hits":           decision.Hits,
				"recall_source":  source,
			})
			return
		}
	}
	if decision.Decision == "skip" {
		l.emitStatus("gate", "无需检索历史，跳过")
	}
	l.emit("gate", map[string]any{
		"decision": decision.Decision,
		"reason":   decision.Reason,
		"hits":     decision.Hits,
	})
}
