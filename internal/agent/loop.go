package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	ctxfrag "github.com/ghsemail/GeeGooAgent/internal/context"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/playbookexec"
	"github.com/ghsemail/GeeGooAgent/internal/prompt"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

const (
	defaultMaxToolRounds   = 80
	defaultToolMaxParallel = 4
	defaultToolTimeout     = 120 * time.Second
)

// Loop runs plan → act → observe for one chat turn.
type Loop struct {
	gateway        *llm.Gateway
	tools          *ToolExec
	maxToolRounds  int
	onProgress     runtime.ProgressFunc
	mem            memport.Port
	skillLoader    *procedural.Loader
	maxSkills      int
	skillTools     SkillToolExpander
	eventBus       tools.EventEmitter
	ranker         cognition.Ranker
	evaluator      cognition.Evaluator
	planPolicy     cognition.PlanPolicy
	planner        cognition.Planner
	evalMaxRetries int
	gateProvider   llm.Provider
	gatePolicy     llm.Policy
	retrievalTopK  int
	playbookRouter *playbookexec.Router
}

// NewLoop creates an agent loop.
func NewLoop(gateway *llm.Gateway, executor *runtime.Executor) *Loop {
	d := cognition.Defaults()
	return &Loop{
		gateway:       gateway,
		tools:         NewToolExec(executor),
		maxToolRounds: defaultMaxToolRounds,
		ranker:        d.Ranker,
		evaluator:     d.Evaluator,
		planPolicy:    d.PlanPolicy,
		planner:       d.Planner,
		mem:           memport.Noop(),
	}
}

// ToolExec returns the shared tool dispatcher (also used by workflow).
func (l *Loop) ToolExec() *ToolExec {
	if l == nil {
		return nil
	}
	return l.tools
}

// SetMaxToolRounds sets the per-turn LLM↔tool iteration cap (config max_steps).
func (l *Loop) SetMaxToolRounds(n int) {
	if l == nil {
		return
	}
	if n <= 0 {
		n = defaultMaxToolRounds
	}
	if n > 90 {
		n = 90
	}
	l.maxToolRounds = n
}

// SetToolMaxParallel caps concurrent tool executions per LLM round.
func (l *Loop) SetToolMaxParallel(n int) {
	if l != nil && l.tools != nil {
		l.tools.SetMaxParallel(n)
	}
}

// SetToolTimeout bounds a single tool invocation.
func (l *Loop) SetToolTimeout(d time.Duration) {
	if l != nil && l.tools != nil {
		l.tools.SetTimeout(d)
	}
}

// SetGateway swaps the LLM gateway (e.g. after /think or /model).
func (l *Loop) SetGateway(gateway *llm.Gateway) {
	l.gateway = gateway
}

// SetCompressor wires optional context compaction (Memory port adapter).
func (l *Loop) SetCompressor(c *prompt.Compressor) {
	l.SetMemory(memory.NewAdapter(memory.AdapterConfig{Compressor: c}))
}

// SetSkillLoader wires procedural memory (SKILL.md keyword match).
func (l *Loop) SetSkillLoader(loader *procedural.Loader, maxSkills int) {
	if l == nil {
		return
	}
	l.skillLoader = loader
	if maxSkills <= 0 {
		maxSkills = 2
	}
	l.maxSkills = maxSkills
}

// SkillToolExpander adds opt-in tool schemas after a skill match (e.g. knowledge-base).
type SkillToolExpander func(skillNames []string) []llm.ToolSchema

// SetSkillToolExpander wires per-turn schema expansion from matched skills.
func (l *Loop) SetSkillToolExpander(fn SkillToolExpander) {
	if l == nil {
		return
	}
	l.skillTools = fn
}

// SetMemory replaces the Memory port (compress / recall / store).
func (l *Loop) SetMemory(m memport.Port) {
	if l == nil {
		return
	}
	if m != nil {
		l.mem = m
	} else {
		l.mem = memport.Noop()
	}
}

// SetRetrievalGate wires the Waku-style LLM retrieval gate (auxiliary model).
func (l *Loop) SetRetrievalGate(provider llm.Provider, policy llm.Policy, topK int) {
	if l == nil {
		return
	}
	l.gateProvider = provider
	l.gatePolicy = policy
	if topK > 0 {
		l.retrievalTopK = topK
	}
}

// SetProgress wires live step output (geegoo chat verbose UI).
func (l *Loop) SetProgress(fn runtime.ProgressFunc) {
	l.onProgress = fn
}

// SetApproval wires interactive confirmation for mutating tools.
func (l *Loop) SetApproval(fn runtime.ApprovalFunc) {
	if l != nil && l.tools != nil {
		l.tools.SetApproval(fn)
	}
}

// SetPlanGate enables plan_proposed events before mutating tool approval.
func (l *Loop) SetPlanGate(v bool) {
	if l != nil && l.tools != nil {
		l.tools.SetPlanGate(v)
	}
}

// SetDelegateMaxParallel caps concurrent delegate_task calls per round.
func (l *Loop) SetDelegateMaxParallel(n int) {
	if l != nil && l.tools != nil {
		l.tools.SetDelegateMaxParallel(n)
	}
}

// SetEventBus wires turn- and tool-level observability (ToolCalled is also emitted by the registry).
func (l *Loop) SetEventBus(bus tools.EventEmitter) {
	l.eventBus = bus
}

// SetEvalMaxRetries caps quality-evaluator driven re-runs per turn (default 0, max 1).
func (l *Loop) SetEvalMaxRetries(n int) {
	if l == nil {
		return
	}
	if n < 0 {
		n = 0
	}
	if n > 1 {
		n = 1
	}
	l.evalMaxRetries = n
}

// SetCognition replaces Ranker / Evaluator / PlanPolicy. Nil fields keep current values.
func (l *Loop) SetCognition(b cognition.Bundle) {
	if l == nil {
		return
	}
	if b.Ranker != nil {
		l.ranker = b.Ranker
	}
	if b.Evaluator != nil {
		l.evaluator = b.Evaluator
	}
	if b.PlanPolicy != nil {
		l.planPolicy = b.PlanPolicy
	}
	if b.Planner != nil {
		l.planner = b.Planner
	}
}

func (l *Loop) effectivePlanner() cognition.Planner {
	if l != nil && l.planner != nil {
		return l.planner
	}
	if l != nil && l.gateProvider != nil {
		return cognition.IntentPlanner{Rules: cognition.RulePlanner{}, LLM: l.gateProvider}
	}
	return cognition.RulePlanner{}
}

func (l *Loop) effectivePlanPolicy() cognition.PlanPolicy {
	if l != nil && l.planPolicy != nil {
		return l.planPolicy
	}
	return cognition.DefaultPlanPolicy{}
}

func (l *Loop) effectiveEvaluator() cognition.Evaluator {
	if l != nil && l.evaluator != nil {
		return l.evaluator
	}
	return cognition.AcceptAllEvaluator{}
}

func (l *Loop) effectiveRanker() cognition.Ranker {
	if l != nil && l.ranker != nil {
		return l.ranker
	}
	return cognition.IdentityRanker{}
}

// RankItems applies the injected Ranker (Kernel hook for recall/snippet ordering).
func (l *Loop) RankItems(ctx context.Context, items []cognition.RankItem) ([]cognition.RankItem, error) {
	return l.effectiveRanker().Rank(ctx, items)
}

func (l *Loop) runEvaluator(ctx context.Context, session *runtime.Session, result runtime.TurnResult) cognition.EvalResult {
	eval := l.effectiveEvaluator()
	res, err := eval.Evaluate(ctx, cognition.EvalInput{
		SessionID:     session.ID,
		AssistantText: result.AssistantText,
		Failed:        result.Failed,
	})
	if err != nil {
		return cognition.EvalResult{Accept: true}
	}
	return res
}

func (l *Loop) emitEvalResult(res cognition.EvalResult) {
	if res.Accept {
		return
	}
	l.emit("cognition_eval", map[string]any{
		"accept": res.Accept, "retry_suggested": res.RetrySuggested, "reason": res.Reason,
	})
}

func (l *Loop) evaluateTurn(ctx context.Context, session *runtime.Session, result runtime.TurnResult) {
	l.emitEvalResult(l.runEvaluator(ctx, session, result))
}

func (l *Loop) tryEvalRetry(
	ctx context.Context,
	session *runtime.Session,
	messages *[]llm.Message,
	result runtime.TurnResult,
	retriesLeft *int,
) bool {
	if l == nil || retriesLeft == nil || *retriesLeft <= 0 || result.Failed || result.PlanPending {
		l.evaluateTurn(ctx, session, result)
		return false
	}
	res := l.runEvaluator(ctx, session, result)
	if res.Accept || !res.RetrySuggested {
		l.emitEvalResult(res)
		return false
	}
	*retriesLeft--
	l.emit("eval_retry", map[string]any{
		"reason": res.Reason, "remaining": *retriesLeft,
	})
	l.emitEvalResult(res)
	hint := res.Reason
	if hint == "" {
		hint = "请根据工具结果改进回答，避免空泛或遗漏关键事实。"
	}
	session.AppendMessage(llm.Message{
		Role:    llm.RoleUser,
		Content: "[质量评估] " + hint + " 请重新组织回答。",
	})
	*messages = session.LLMMessages()
	return true
}

func (l *Loop) emit(event string, data map[string]any) {
	if l.onProgress != nil {
		l.onProgress(event, data)
	}
}

func (l *Loop) emitBus(event string, data map[string]any) {
	if l.eventBus != nil {
		l.eventBus.Emit(event, data)
	}
}

func (l *Loop) RunTurn(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
) runtime.TurnResult {
	if ctx == nil {
		ctx = context.Background()
	}
	toolCtx.Ctx = ctx
	if toolCtx.EventBus == nil && l.eventBus != nil {
		toolCtx.EventBus = l.eventBus
	}
	if toolCtx.Interactive {
		schemas = filterInteractiveSchemas(schemas)
	}

	session.AppendMessage(llm.Message{Role: llm.RoleUser, Content: userText})
	records := []runtime.StepRecord{}

	l.emit("turn_start", map[string]any{"user_text": userText})
	l.emitBus("TurnStarted", map[string]any{
		"session_id": session.ID, "user_text": userText,
	})
	l.emitStatus("received", "已收到消息，准备处理")
	return l.runPreparedTurn(ctx, session, userText, toolCtx, schemas, records)
}

func (l *Loop) runPreparedTurn(
	ctx context.Context,
	session *runtime.Session,
	userText string,
	toolCtx tools.Context,
	schemas []llm.ToolSchema,
	records []runtime.StepRecord,
) runtime.TurnResult {
	messages := session.LLMMessages()

	policy := l.effectivePlanPolicy()
	if session.PendingPlan != nil {
		messages = session.LLMMessages()
		if policy.IsApproval(userText) {
			result := l.resumePendingPlan(ctx, session, &messages, toolCtx, schemas, &records)
			l.evaluateTurn(ctx, session, result)
			return result
		}
		if policy.IsRejection(userText) {
			result := l.cancelPendingPlan(session)
			result.StepRecords = records
			l.evaluateTurn(ctx, session, result)
			return result
		}
		session.PendingPlan = nil
	}

	l.emitStatus("plan", "正在判断本轮意图…")
	planStarted := time.Now()
	turnPlan := l.effectivePlanner().Plan(cognition.PlanInput{
		Ctx:        ctx,
		UserText:   userText,
		LastDomain: cognition.Domain(session.LastTurnDomain),
	})
	planMS := time.Since(planStarted).Milliseconds()
	session.LastTurnDomain = string(turnPlan.Domain)
	session.LastTurnMode = string(turnPlan.Mode)
	session.LastTurnSOP = turnPlan.ShouldRunDomainSOP()
	session.LastTurnToolsAllow = append([]string(nil), turnPlan.ToolsAllow...)
	l.emit("turn_plan", map[string]any{
		"domain":     string(turnPlan.Domain),
		"act":        turnPlan.Act,
		"mode":       string(turnPlan.Mode),
		"reason":     turnPlan.Reason,
		"skills":     turnPlan.Skills,
		"tools":      turnPlan.ToolsAllow,
		"confidence": turnPlan.Confidence,
	})
	l.emitStatus("plan", fmt.Sprintf("判断：%s/%s（%dms）", turnPlan.Domain, turnPlan.Mode, planMS))

	procFrag, matchedSkills := l.loadPlanSkills(turnPlan, &records)
	var gateFrag ctxfrag.Fragment
	if ShouldSkipRetrievalGate(matchedSkills, turnPlan, userText) {
		reason := skipRetrievalReason(matchedSkills, turnPlan, userText)
		if turnPlan.Mode == cognition.ModeClarify {
			l.recordInjectionStep(&records, "gate", "decision=skip · reason="+reason)
			l.emit("gate", map[string]any{"decision": "skip", "reason": reason})
		} else {
			if reason == "tool-first playbook" {
				l.emitStatus("gate", "工具型技能，跳过记忆检索")
			}
			l.emit("gate", map[string]any{
				"decision": "skip",
				"reason":   reason,
			})
			l.recordInjectionStep(&records, "gate", "decision=skip · reason="+reason)
		}
	} else {
		gateFrag = l.runRetrievalGate(ctx, session, userText, &records)
	}
	dynFrags := []ctxfrag.Fragment{ctxfrag.ClockFragment(clockNow()), turnPlanFragment(turnPlan)}
	if gateFrag != nil && strings.TrimSpace(gateFrag.Render()) != "" {
		dynFrags = append(dynFrags, gateFrag)
	}
	if procFrag != nil && strings.TrimSpace(procFrag.Render()) != "" {
		dynFrags = append(dynFrags, procFrag)
	}
	l.applyDynamicFragments(session, dynFrags, &records)
	if extra := l.expandSkillSchemas(matchedSkills); len(extra) > 0 {
		schemas = mergeToolSchemas(schemas, extra)
	}
	schemas = cognition.FilterSchemas(schemas, turnPlan)

	if result, handled := l.tryPresetClarify(ctx, session, turnPlan, toolCtx, &records, schemas); handled {
		return result
	}

	if turnPlan.ShouldRunDomainSOP() && l.playbookRouter != nil {
		if result, handled := l.playbookRouter.TryRunFromPlan(ctx, playbookexec.Input{
			Session:       session,
			UserText:      userText,
			MatchedSkills: matchedSkills,
			ToolCtx:       toolCtx,
			StepBase:      session.StepCounter + 1,
			OnProgress:    l.onProgress,
		}, string(turnPlan.Domain)); handled {
			l.evaluateTurn(ctx, session, result)
			return result
		}
	}
	schemas = playbookexec.FilterLegacyBacktestTools(schemas, userText, session)

	messages = session.LLMMessages()
	l.emitStatus("hygiene", "整理会话上下文…")
	messages = l.applyHygiene(ctx, session, messages)
	evalRetriesLeft := l.evalMaxRetries

	for round := 0; round < l.maxToolRounds; round++ {
		if err := ctx.Err(); err != nil {
			return l.failTurn(ctx, session, err, records)
		}
		done, result := l.runRound(ctx, session, &messages, toolCtx, schemas, round, &records)
		if done {
			if l.tryEvalRetry(ctx, session, &messages, result, &evalRetriesLeft) {
				continue
			}
			if !result.Failed {
				l.emitBus("TurnCompleted", map[string]any{
					"session_id": session.ID, "steps": len(result.StepRecords),
				})
			}
			result.StepRecords = records
			return result
		}
	}

	msg := l.finishBudgetExhausted(ctx, session, messages, records)
	l.evaluateTurn(ctx, session, msg)
	return msg
}

func (l *Loop) failTurn(ctx context.Context, session *runtime.Session, err error, records []runtime.StepRecord) runtime.TurnResult {
	msg := fmt.Sprintf("已中断: %v", err)
	l.emit("error", map[string]any{"message": msg})
	l.emitBus("TurnFailed", map[string]any{
		"session_id": session.ID, "error": err.Error(),
	})
	return runtime.TurnResult{AssistantText: msg, Failed: true, Error: err.Error(), StepRecords: records}
}
