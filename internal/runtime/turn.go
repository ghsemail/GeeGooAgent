package runtime

// TurnResult is the outcome of one user turn.
type TurnResult struct {
	AssistantText string
	Failed        bool
	Error         string
	StepRecords   []StepRecord
}
