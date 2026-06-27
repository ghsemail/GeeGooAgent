package runtime

import "time"

// StepRecord is one plan/tool/reply step in a chat turn.
type StepRecord struct {
	Step       int
	Timestamp  time.Time
	Kind       string
	ToolName   string
	ToolStatus string
	Summary    string
}
