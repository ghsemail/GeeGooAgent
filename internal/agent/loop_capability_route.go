package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/agent/decision"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func (l *Loop) consultCapabilityRoute(ctx context.Context, session *runtime.Session, userText string, records *[]runtime.StepRecord) {
	if !l.decisionEnabled() || l.skillLoader == nil {
		return
	}
	catalog := l.skillLoader.ChatCatalog()
	if len(catalog) == 0 {
		return
	}
	choices := []decision.Choice{{ID: "_none", Label: "no skill preference"}}
	for _, e := range catalog {
		choices = append(choices, decision.Choice{ID: e.ID, Label: truncateCatalogLabel(e.Description)})
	}
	sid := ""
	if session != nil {
		sid = session.ID
	}
	l.emitDecisionConsult(ctx, decision.Question{
		ID:        decisionQuestionID(decision.KindCapabilityRoute, 0, "pre_round"),
		Kind:      decision.KindCapabilityRoute,
		Step:      0,
		SessionID: sid,
		Choices:   choices,
		Context: map[string]any{
			"user_text":     truncateGateText(userText),
			"catalog_count": len(catalog),
			"skill_injection": string(l.effectiveSkillInjection()),
		},
	})
	if records != nil {
		l.recordInjectionStep(records, "capability_route", fmt.Sprintf("shadow consult · %d catalog skills", len(catalog)))
	}
}

func (l *Loop) effectiveSkillInjection() procedural.SkillInjectionMode {
	if l == nil || l.skillInjection == "" {
		return procedural.SkillInjectionCatalog
	}
	return l.skillInjection
}

func truncateCatalogLabel(desc string) string {
	desc = strings.TrimSpace(desc)
	r := []rune(desc)
	if len(r) <= 48 {
		return desc
	}
	return string(r[:48]) + "…"
}

func truncateGateText(text string) string {
	text = strings.TrimSpace(text)
	r := []rune(text)
	if len(r) <= 80 {
		return text
	}
	return string(r[:80]) + "…"
}
