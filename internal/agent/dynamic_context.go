package agent

import (
	"fmt"
	"strings"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

const dynamicContextMaxBytes = ctxfrag.DefaultComposeMaxBytes

// applyDynamicFragments composes turn-level fragments and injects one system block before the latest user message.
func (l *Loop) applyDynamicFragments(session *runtime.Session, fragments []ctxfrag.Fragment, records *[]runtime.StepRecord) {
	if l == nil || session == nil || len(fragments) == 0 {
		return
	}
	text, applied, dropped := ctxfrag.Composer{MaxBytes: dynamicContextMaxBytes}.Compose(fragments)
	text = strings.TrimSpace(text)
	if text == "" {
		return
	}
	injectBeforeLastUser(session, llm.Message{
		Role:    llm.RoleSystem,
		Content: text,
	})
	l.emit("context_fragment_applied", map[string]any{
		"applied": ctxfrag.KindsToStrings(applied),
		"dropped": ctxfrag.KindsToStrings(dropped),
		"bytes":   len([]byte(text)),
	})
	summary := fmt.Sprintf("context fragments: applied=%s", strings.Join(ctxfrag.KindsToStrings(applied), ","))
	if len(dropped) > 0 {
		summary += " dropped=" + strings.Join(ctxfrag.KindsToStrings(dropped), ",")
	}
	l.recordInjectionStep(records, "context_fragment", summary)
}

// injectBeforeLastUser inserts a message immediately before the trailing user turn.
func injectBeforeLastUser(session *runtime.Session, mem llm.Message) {
	if session == nil || strings.TrimSpace(mem.Content) == "" {
		return
	}
	n := len(session.Messages)
	if n >= 1 && session.Messages[n-1].Role == llm.RoleUser {
		session.Messages = append(session.Messages[:n-1], append([]llm.Message{mem}, session.Messages[n-1:]...)...)
		return
	}
	session.AppendMessage(mem)
}

// appendEphemeralUserFragments composes ephemeral fragments into a temporary user message (not persisted).
func appendEphemeralUserFragments(messages []llm.Message, fragments []ctxfrag.Fragment, maxBytes int) []llm.Message {
	if len(fragments) == 0 {
		return messages
	}
	if maxBytes <= 0 {
		maxBytes = 4096
	}
	text, _, _ := ctxfrag.Composer{MaxBytes: maxBytes}.Compose(fragments)
	text = strings.TrimSpace(text)
	if text == "" {
		return messages
	}
	out := make([]llm.Message, len(messages)+1)
	copy(out, messages)
	out[len(messages)] = llm.Message{Role: llm.RoleUser, Content: text}
	return out
}

func withHookInjectFragments(messages []llm.Message, session *runtime.Session, l *Loop) []llm.Message {
	if session == nil || l == nil {
		return messages
	}
	inject := strings.TrimSpace(session.PendingHookInject)
	if inject == "" {
		return messages
	}
	if l.tools == nil || !l.tools.FragmentInjectEnabled() {
		return appendEphemeralUserFragments(messages, []ctxfrag.Fragment{ctxfrag.HookInjectFragment(inject)}, 4096)
	}
	frag := ctxfrag.HookInjectFragment(inject)
	composed, applied, dropped := ctxfrag.Composer{MaxBytes: 4096}.Compose([]ctxfrag.Fragment{frag})
	session.PendingHookInject = ""
	if composed == "" {
		return messages
	}
	l.emit("context_fragment_applied", map[string]any{
		"source": "hook", "kinds": ctxfrag.KindsToStrings(applied), "dropped": ctxfrag.KindsToStrings(dropped),
	})
	return appendEphemeralUserFragments(messages, []ctxfrag.Fragment{ctxfrag.StaticFragment{
		K: ctxfrag.KindHookInject, Text: composed, Prio: 40,
	}}, 4096)
}
