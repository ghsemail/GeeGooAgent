package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
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

func recallFragmentLabel(source string) string {
	switch source {
	case "facts":
		return "semantic facts (FTS)"
	case "episodic":
		return "episodic memory (FTS)"
	case "facts+episodic":
		return "semantic facts + episodic memory"
	default:
		return "long-term memory (FTS)"
	}
}

func (l *Loop) recordInjectionStep(records *[]runtime.StepRecord, kind, summary string) {
	if records == nil || strings.TrimSpace(summary) == "" {
		return
	}
	*records = append(*records, runtime.StepRecord{
		Step:      0,
		Timestamp: time.Now().UTC(),
		Kind:      kind,
		Summary:   strings.TrimSpace(summary),
	})
}

func gateDecisionSummary(d RetrievalGateDecision, source string) string {
	parts := []string{
		fmt.Sprintf("decision=%s", d.Decision),
		fmt.Sprintf("hits=%d", d.Hits),
	}
	if q := strings.TrimSpace(d.Query); q != "" {
		parts = append(parts, "query="+q)
	}
	if source != "" {
		parts = append(parts, "source="+source)
	}
	if r := strings.TrimSpace(d.Reason); r != "" {
		parts = append(parts, r)
	}
	return strings.Join(parts, " · ")
}

func (l *Loop) runRetrievalGate(ctx context.Context, session *runtime.Session, userText string, records *[]runtime.StepRecord) ctxfrag.Fragment {
	l.emitStatus("gate", "检查是否需要检索长期记忆…")
	gate := retrievalgate.ShouldRetrieve(ctx, l.gateProvider, l.gatePolicy, userText)
	decision := RetrievalGateDecision{
		Decision: "skip",
		Reason:   gate.Reason,
		Query:    gate.Query,
	}
	source := ""
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
			Kind:      memport.RecallSession,
			Query:     query,
			UserID:    session.UserID,
			SessionID: session.ID,
			Limit:     topK,
		})
		if err != nil {
			decision.Reason = "recall error: " + err.Error()
		} else if len(res.Hits) == 0 {
			decision.Reason = "retrieve=yes but no matching memories"
		} else {
			source = "facts+episodic"
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
			l.recordInjectionStep(records, "gate", gateDecisionSummary(decision, source))
			if block := formatGateMemory(res.Hits); block != "" {
				return ctxfrag.RecallFragment(recallFragmentLabel(source), block)
			}
			return nil
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
	l.recordInjectionStep(records, "gate", gateDecisionSummary(decision, source))
	return nil
}

func (l *Loop) runProceduralMemory(session *runtime.Session, userText string, records *[]runtime.StepRecord) (ctxfrag.Fragment, []string) {
	if l == nil || l.skillLoader == nil || session == nil {
		return nil, nil
	}
	maxSkills := l.maxSkills
	if maxSkills <= 0 {
		maxSkills = 2
	}
	matched := l.skillLoader.Match(userText, maxSkills)
	matched = l.skillLoader.PrioritizeSignalProbePlaybook(userText, matched, maxSkills)
	matched = l.skillLoader.PrioritizeBacktestRunPlaybook(userText, matched, maxSkills)
	if len(matched) == 0 {
		return nil, nil
	}
	block := procedural.Format(matched)
	names := skillNames(matched)
	if block == "" {
		return nil, names
	}
	l.emitStatus("gate", fmt.Sprintf("加载 %d 个相关技能 (procedural)", len(matched)))
	l.emit("memory.procedural", map[string]any{
		"skills": len(matched),
		"names":  names,
	})
	l.recordInjectionStep(records, "context_inject", fmt.Sprintf("procedural skills: %s", strings.Join(names, ", ")))
	return ctxfrag.ProceduralSkillFragment(block), names
}

func (l *Loop) expandSkillSchemas(skillNames []string) []llm.ToolSchema {
	if l == nil || l.skillTools == nil || len(skillNames) == 0 {
		return nil
	}
	return l.skillTools(skillNames)
}

func mergeToolSchemas(base, extra []llm.ToolSchema) []llm.ToolSchema {
	if len(extra) == 0 {
		return base
	}
	seen := map[string]struct{}{}
	out := make([]llm.ToolSchema, 0, len(base)+len(extra))
	for _, s := range base {
		if s.Name == "" {
			continue
		}
		if _, ok := seen[s.Name]; ok {
			continue
		}
		seen[s.Name] = struct{}{}
		out = append(out, s)
	}
	for _, s := range extra {
		if s.Name == "" {
			continue
		}
		if _, ok := seen[s.Name]; ok {
			continue
		}
		seen[s.Name] = struct{}{}
		out = append(out, s)
	}
	return out
}

func skillNames(matched []procedural.Skill) []string {
	out := make([]string, len(matched))
	for i, sk := range matched {
		out[i] = sk.Name
	}
	return out
}
