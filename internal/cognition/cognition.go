// Package cognition holds replaceable intelligence strategies for the Agent Kernel.
//
// Kernel (internal/agent Loop) owns the control plane; this package owns Planner/
// Ranker/Evaluator-style policy surfaces. Default implementations are pure Go and
// preserve today's behavior. Optional Advisor sidecars may implement the same
// interfaces later (suggestion-only; no tool/state ownership).
package cognition

import "context"

// Bundle groups cognition strategies injected into the Kernel.
type Bundle struct {
	Ranker     Ranker
	Evaluator  Evaluator
	PlanPolicy PlanPolicy
}

// Defaults returns Go default strategies (behavior-preserving).
func Defaults() Bundle {
	return Bundle{
		Ranker:     IdentityRanker{},
		Evaluator:  AcceptAllEvaluator{},
		PlanPolicy: DefaultPlanPolicy{},
	}
}

// RankItem is a candidate snippet or memory hit for ranking.
type RankItem struct {
	ID    string
	Text  string
	Score float64
	Meta  map[string]any
}

// Ranker reorders or re-scores candidates. Default: identity order.
type Ranker interface {
	Rank(ctx context.Context, items []RankItem) ([]RankItem, error)
}

// IdentityRanker leaves order unchanged.
type IdentityRanker struct{}

// Rank returns items as-is.
func (IdentityRanker) Rank(_ context.Context, items []RankItem) ([]RankItem, error) {
	if items == nil {
		return nil, nil
	}
	out := make([]RankItem, len(items))
	copy(out, items)
	return out, nil
}

// EvalInput is a read-only snapshot for post-turn judgment.
type EvalInput struct {
	SessionID     string
	AssistantText string
	Failed        bool
	ToolNames     []string
}

// EvalResult is an advisory judgment; Kernel decides whether to act.
type EvalResult struct {
	Accept         bool
	RetrySuggested bool
	Reason         string
}

// Evaluator judges turn quality. Default: always accept.
type Evaluator interface {
	Evaluate(ctx context.Context, in EvalInput) (EvalResult, error)
}

// AcceptAllEvaluator always accepts and never suggests retry.
type AcceptAllEvaluator struct{}

// Evaluate always accepts.
func (AcceptAllEvaluator) Evaluate(_ context.Context, _ EvalInput) (EvalResult, error) {
	return EvalResult{Accept: true}, nil
}

// PlanHoldInput drives mutating-tool hold decisions.
type PlanHoldInput struct {
	GateEnabled   bool
	Interactive   bool
	Approved      bool
	MutatingCount int
}

// ProposedCall is a tool call summary for plan_proposed payloads.
type ProposedCall struct {
	Name      string
	Arguments map[string]any
}

// PlanPolicy decides plan-hold / approval text for mutating tools.
type PlanPolicy interface {
	ShouldHold(in PlanHoldInput) bool
	IsApproval(text string) bool
	IsRejection(text string) bool
	HoldMessage(planText string) string
	ProposedPayload(calls []ProposedCall) map[string]any
}
