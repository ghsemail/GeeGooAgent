package taskflow

import (
	"encoding/json"

	"github.com/ghsemail/GeeGooAgent/internal/runtime"
)

// LoadFromSession decodes flow state from runtime.Session.ActiveFlowJSON.
func LoadFromSession(session *runtime.Session) *Flow {
	if session == nil || len(session.ActiveFlowJSON) == 0 {
		return nil
	}
	var flow Flow
	if err := json.Unmarshal(session.ActiveFlowJSON, &flow); err != nil {
		return nil
	}
	if flow.RunID == "" {
		return nil
	}
	return &flow
}

// SaveToSession encodes flow state onto runtime.Session.ActiveFlowJSON.
func SaveToSession(session *runtime.Session, flow *Flow) {
	if session == nil {
		return
	}
	if flow == nil {
		session.ActiveFlowJSON = nil
		return
	}
	raw, err := json.Marshal(flow)
	if err != nil {
		return
	}
	session.ActiveFlowJSON = raw
}
