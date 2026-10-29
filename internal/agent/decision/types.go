package decision

import "context"

// Kind marks which ReAct decision point is being consulted.
type Kind string

const (
	KindEndTurn  Kind = "end_turn"
	KindToolPlan Kind = "tool_plan"
	KindPlanGate Kind = "plan_gate"
	KindClarify  Kind = "clarify"
)

// Mode controls whether the hook may take over control from the main LLM path.
type Mode string

const (
	ModeOff     Mode = "off"
	ModeShadow  Mode = "shadow"
	ModeReplace Mode = "replace" // reserved: Coordinator never takeovers plan_gate / free text
	ModePaper   Mode = "paper"   // reserved: same as shadow until paper eval wiring exists
)

// Who records which actor produced the winning path (telemetry only in shadow).
type Who string

const (
	WhoJev       Who = "jev"
	WhoLLM       Who = "llm"
	WhoFallback  Who = "fallback"
)

// Choice is one typed option for a decision question (no free-form tool args).
type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label,omitempty"`
}

// Question is a structured multiple-choice decision inside one ReAct step.
type Question struct {
	ID        string         `json:"question_id"`
	Kind      Kind           `json:"kind"`
	Step      int            `json:"step"`
	SessionID string         `json:"session_id,omitempty"`
	Choices   []Choice       `json:"choices"`
	Context   map[string]any `json:"context,omitempty"`
}

// Answer is the hook outcome. ChoiceID must match one of Question.Choices[].ID when set.
type Answer struct {
	QuestionID string  `json:"question_id"`
	ChoiceID   string  `json:"answer"`
	Confidence float64 `json:"confidence"`
	Who        Who     `json:"who"`
	Reason     string  `json:"reason,omitempty"`
}

// Hook decides among typed choices (Jev / stub). Must not emit user-facing prose or tool JSON.
type Hook interface {
	Decide(ctx context.Context, q Question) (Answer, error)
}

// ProgressFunc mirrors Loop SSE progress callbacks.
type ProgressFunc func(event string, data map[string]any)
