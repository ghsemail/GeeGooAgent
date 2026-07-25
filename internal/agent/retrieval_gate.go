package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// RetrievalGateDecision is the outcome of the pre-turn memory gate.
type RetrievalGateDecision struct {
	Decision string // skip | retrieve
	Reason   string
	Hits     int
	Query    string
}

func countRecallKinds(hits []memport.RecallHit) (facts, episodes int) {
	for _, h := range hits {
		if h.Data == nil {
			continue
		}
		switch h.Data["kind"] {
		case "fact":
			facts++
		case "episode":
			episodes++
		}
	}
	return facts, episodes
}

func formatGateMemory(hits []memport.RecallHit) string {
	if len(hits) == 0 {
		return ""
	}
	var b strings.Builder
	for i, h := range hits {
		if i >= 7 {
			break
		}
		if i > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "- %s", strings.TrimSpace(h.Snippet))
	}
	return b.String()
}

func (l *Loop) injectGateMemory(session *runtime.Session, block, source string) {
	if session == nil || strings.TrimSpace(block) == "" {
		return
	}
	label := "long-term memory (FTS)"
	switch source {
	case "facts":
		label = "semantic facts (FTS)"
	case "episodic":
		label = "episodic memory (FTS)"
	case "facts+episodic":
		label = "semantic facts + episodic memory"
	}
	mem := llm.Message{
		Role: llm.RoleSystem,
		Content: fmt.Sprintf("Retrieved from %s. Use only if relevant:\n%s", label, block),
	}
	n := len(session.Messages)
	if n >= 1 && session.Messages[n-1].Role == llm.RoleUser {
		session.Messages = append(session.Messages[:n-1], append([]llm.Message{mem}, session.Messages[n-1:]...)...)
		return
	}
	session.AppendMessage(mem)
}

func (l *Loop) runRetrievalGate(ctx context.Context, session *runtime.Session, userText string) {
	l.emitStatus("gate", "检查是否需要检索长期记忆…")
	gate := retrievalgate.ShouldRetrieve(ctx, l.gateProvider, l.gatePolicy, userText)
	decision := RetrievalGateDecision{
		Decision: "skip",
		Reason:   gate.Reason,
		Query:    gate.Query,
	}
	if gate.Retrieve && l.mem != nil {
		topK := l.retrievalTopK
		if topK <= 0 {
			topK = 4
		}
		query := strings.TrimSpace(gate.Query)
		if query == "" {
			query = userText
		}
		res, err := l.mem.Recall(ctx, memport.RecallQuery{
			Kind:   memport.RecallSession,
			Query:  query,
			UserID: session.UserID,
			Limit:  topK,
		})
		if err != nil {
			decision.Reason = "recall error: " + err.Error()
		} else if len(res.Hits) == 0 {
			decision.Reason = "retrieve=yes but no matching memories"
		} else {
			source := "facts+episodic"
			if res.Data != nil {
				if v, ok := res.Data["recall_source"].(string); ok && v != "" {
					source = v
				}
			}
			decision.Decision = "retrieve"
			decision.Hits = len(res.Hits)
			if decision.Reason == "" {
				decision.Reason = fmt.Sprintf("gate matched %d memories", len(res.Hits))
			}
			l.emitStatus("gate", fmt.Sprintf("检索到 %d 条相关记忆（%s）", len(res.Hits), source))
			if block := formatGateMemory(res.Hits); block != "" {
				l.injectGateMemory(session, block, source)
			}
			factsN, episodesN := countRecallKinds(res.Hits)
			l.emit("gate", map[string]any{
				"decision":      decision.Decision,
				"reason":        decision.Reason,
				"hits":          decision.Hits,
				"facts":         factsN,
				"episodes":      episodesN,
				"query":         query,
				"recall_source": source,
			})
			return
		}
	}
	if !gate.Retrieve {
		l.emitStatus("gate", "无需检索记忆，跳过")
	} else {
		l.emitStatus("gate", "需要检索但未命中记忆")
	}
	l.emit("gate", map[string]any{
		"decision": decision.Decision,
		"reason":   decision.Reason,
		"hits":     decision.Hits,
		"query":    decision.Query,
	})
}

func (l *Loop) runProceduralMemory(session *runtime.Session, userText string) {
	if l == nil || l.skillLoader == nil || session == nil {
		return
	}
	maxSkills := l.maxSkills
	if maxSkills <= 0 {
		maxSkills = 2
	}
	matched := l.skillLoader.Match(userText, maxSkills)
	if len(matched) == 0 {
		return
	}
	block := procedural.Format(matched)
	if block == "" {
		return
	}
	l.emitStatus("gate", fmt.Sprintf("加载 %d 个相关技能 (procedural)", len(matched)))
	l.emit("memory.procedural", map[string]any{
		"skills": len(matched),
		"names":  skillNames(matched),
	})
	injectProceduralMemory(session, block)
}

func skillNames(matched []procedural.Skill) []string {
	out := make([]string, len(matched))
	for i, sk := range matched {
		out[i] = sk.Name
	}
	return out
}

func injectProceduralMemory(session *runtime.Session, block string) {
	if session == nil || strings.TrimSpace(block) == "" {
		return
	}
	mem := llm.Message{
		Role:    llm.RoleSystem,
		Content: "Relevant skill instructions (procedural memory). Follow only if applicable:\n" + block,
	}
	n := len(session.Messages)
	if n >= 1 && session.Messages[n-1].Role == llm.RoleUser {
		session.Messages = append(session.Messages[:n-1], append([]llm.Message{mem}, session.Messages[n-1:]...)...)
		return
	}
	session.AppendMessage(mem)
}
