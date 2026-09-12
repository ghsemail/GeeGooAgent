package runtimeapi

import (
	"encoding/json"
	"strings"
)

// evalRunVisibleToUser decides whether an eval run row is visible in Cockpit history.
// Logged-in operators also see legacy anonymous runs (user_id="").
func evalRunVisibleToUser(rowUserID, requestUserID string) bool {
	rowUserID = strings.TrimSpace(rowUserID)
	requestUserID = strings.TrimSpace(requestUserID)
	if requestUserID == "" {
		return rowUserID == ""
	}
	return rowUserID == requestUserID || rowUserID == ""
}

// evalRunsWhereClause returns SQL filter + args for listing/accessing eval runs.
func evalRunsWhereClause(requestUserID string) (clause string, args []any) {
	requestUserID = strings.TrimSpace(requestUserID)
	if requestUserID == "" {
		return "user_id = ''", nil
	}
	return "user_id = ? OR user_id = ''", []any{requestUserID}
}

func sessionsJSONForEval(sessionID string) string {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "[]"
	}
	raw, _ := json.Marshal([]map[string]string{
		{"label": "Dock", "session_id": sessionID},
	})
	return string(raw)
}

func normalizeEvalRunStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass":
		return "passed"
	case "fail", "error", "cancelled":
		return "failed"
	default:
		return strings.TrimSpace(status)
	}
}
