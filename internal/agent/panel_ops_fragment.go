package agent

import (
	"strings"

	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

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
