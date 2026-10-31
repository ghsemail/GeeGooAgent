package agent

import (
	"strings"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

func lockedSkillFragment(session *runtime.Session) ctxfrag.Fragment {
	if session == nil {
		return ctxfrag.StaticFragment{}
	}
	text := strings.TrimSpace(runtime.LockedSkillTurnHint(session.LockedSkill))
	if text == "" {
		return ctxfrag.StaticFragment{}
	}
	return ctxfrag.StaticFragment{
		K:    ctxfrag.KindSystemRules,
		Text: text,
		Prio: 25,
	}
}

func signalDiagnosePanelOpsFragment(session *runtime.Session) ctxfrag.Fragment {
	text := runtime.PanelOpsTurnHint(session)
	text = strings.TrimSpace(text)
	if text == "" {
		return ctxfrag.StaticFragment{}
	}
	return ctxfrag.StaticFragment{
		K:    ctxfrag.KindSystemRules,
		Text: text,
		Prio: 26,
	}
}
