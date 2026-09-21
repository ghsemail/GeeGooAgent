package clarifycontext

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/chatsession"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/sessiontask"
)

const (
	maxDialogueLines   = 8
	maxLineRunes       = 200
	maxSummaryRunes    = 400
	maxWorkingRunes    = 400
	maxRecentSteps     = 4
	maxStepSummary     = 140
)

// DialogueLine is one turn snippet for JEV / fallback LLM.
type DialogueLine struct {
	Role    string `json:"role"`
	Summary string `json:"summary"`
}

// Bundle is the structured context passed to JEV for clarify recommendation.
type Bundle struct {
	ClarifyKind    string            `json:"clarify_kind"`
	Trigger        string            `json:"trigger,omitempty"`
	Question       string            `json:"question"`
	Options        []Option          `json:"options"`
	LastTurn       *TurnSnapshot     `json:"last_turn,omitempty"`
	Dialogue       []DialogueLine    `json:"dialogue,omitempty"`
	SessionSummary string            `json:"session_summary,omitempty"`
	WorkingState   string            `json:"working_state,omitempty"`
	RecentSteps    []string          `json:"recent_tool_summaries,omitempty"`
	Slots          map[string]string `json:"slots,omitempty"`
	// EvalPreference is an optional eval-case hint (clarify_reply), not a forced user utterance.
	EvalPreference string `json:"eval_preference,omitempty"`
}

// Option is one clarify choice with stable id.
type Option struct {
	Index int    `json:"index"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

// TurnSnapshot mirrors persisted turn plan metadata.
type TurnSnapshot struct {
	Domain string `json:"domain,omitempty"`
	Mode   string `json:"mode,omitempty"`
	Act    string `json:"act,omitempty"`
}

// BuildFromChat assembles a decision bundle from session history + clarify prompt.
func BuildFromChat(chat *chatsession.ChatSession, question string, choices []string, trigger string) Bundle {
	b := Bundle{
		ClarifyKind: InferKind(question, choices),
		Trigger:     strings.TrimSpace(trigger),
		Question:    strings.TrimSpace(question),
		Options:     formatOptions(choices),
	}
	if chat == nil {
		return b
	}
	if summary := strings.TrimSpace(chat.Summary); summary != "" {
		b.SessionSummary = truncateRunes(summary, maxSummaryRunes)
	}
	if snap, ok := chatsession.LastTurnPlanFromSession(chat); ok && snap.Domain != "" {
		b.LastTurn = &TurnSnapshot{
			Domain: snap.Domain,
			Mode:   snap.Mode,
			Act:    snap.Act,
		}
	}
	b.Dialogue = dialogueWindow(chat.Messages)
	rt := &runtime.Session{ID: chat.ID, Messages: chat.RuntimeMessages()}
	domain := ""
	if b.LastTurn != nil {
		domain = b.LastTurn.Domain
	}
	if md := sessiontask.BuildState(rt, domain).RenderMarkdown(); md != "" {
		b.WorkingState = truncateRunes(md, maxWorkingRunes)
	}
	b.RecentSteps = recentStepSummaries(chat.StepRecords)
	b.Slots = slotsFromState(sessiontask.BuildState(rt, domain))
	return b
}

func formatOptions(choices []string) []Option {
	out := make([]Option, 0, len(choices))
	for i, c := range choices {
		text := strings.TrimSpace(c)
		if text == "" {
			continue
		}
		id := string(rune('A' + i))
		if i >= 26 {
			id = fmt.Sprintf("%d", i+1)
		}
		out = append(out, Option{Index: i, ID: id, Text: text})
	}
	return out
}

func dialogueWindow(msgs []llm.Message) []DialogueLine {
	if len(msgs) == 0 {
		return nil
	}
	start := 0
	if len(msgs) > maxDialogueLines {
		start = len(msgs) - maxDialogueLines
	}
	out := make([]DialogueLine, 0, maxDialogueLines)
	for _, msg := range msgs[start:] {
		switch msg.Role {
		case llm.RoleUser, llm.RoleAssistant:
		default:
			continue
		}
		text := scrubMessageContent(msg.Content)
		if text == "" {
			continue
		}
		out = append(out, DialogueLine{
			Role:    string(msg.Role),
			Summary: truncateRunes(text, maxLineRunes),
		})
	}
	return out
}

func scrubMessageContent(content string) string {
	s := strings.TrimSpace(content)
	if s == "" {
		return ""
	}
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			break
		}
		end := strings.Index(s, "</think>")
		if end < 0 {
			s = strings.TrimSpace(s[:start])
			break
		}
		s = strings.TrimSpace(s[:start] + s[end+len("</think>"):])
	}
	return strings.TrimSpace(s)
}

func recentStepSummaries(records []chatsession.ChatStepRecord) []string {
	if len(records) == 0 {
		return nil
	}
	start := 0
	if len(records) > maxRecentSteps {
		start = len(records) - maxRecentSteps
	}
	out := make([]string, 0, maxRecentSteps)
	for _, rec := range records[start:] {
		kind := strings.TrimSpace(rec.Kind)
		tool := strings.TrimSpace(rec.ToolName)
		summary := truncateRunes(strings.TrimSpace(rec.Summary), maxStepSummary)
		if summary == "" && kind == "" && tool == "" {
			continue
		}
		line := summary
		if tool != "" {
			line = tool + ": " + summary
		} else if kind != "" && summary == "" {
			line = kind
		}
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func slotsFromState(st sessiontask.State) map[string]string {
	out := map[string]string{}
	if v := strings.TrimSpace(st.Symbol); v != "" {
		out["symbol"] = v
	}
	if v := strings.TrimSpace(st.Strategy); v != "" {
		out["strategy"] = v
	}
	if v := strings.TrimSpace(st.Frequency); v != "" {
		out["frequency"] = v
	}
	if v := strings.TrimSpace(st.Task); v != "" {
		out["task"] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func truncateRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// DecisionJSON returns the payload for JEV (task clarify_recommend).
func (b Bundle) DecisionJSON() (string, error) {
	payload := map[string]any{
		"task":         "clarify_recommend",
		"clarify_kind": b.ClarifyKind,
		"question":     b.Question,
		"options":      b.Options,
	}
	if b.Trigger != "" {
		payload["trigger"] = b.Trigger
	}
	if b.LastTurn != nil {
		payload["last_turn"] = b.LastTurn
	}
	if len(b.Dialogue) > 0 {
		payload["dialogue"] = b.Dialogue
	}
	if b.SessionSummary != "" {
		payload["session_summary"] = b.SessionSummary
	}
	if b.WorkingState != "" {
		payload["working_state"] = b.WorkingState
	}
	if len(b.RecentSteps) > 0 {
		payload["recent_tool_summaries"] = b.RecentSteps
	}
	if len(b.Slots) > 0 {
		payload["slots"] = b.Slots
	}
	if b.EvalPreference != "" {
		payload["eval_preference"] = b.EvalPreference
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
