package agent

import (
	"context"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/cognition"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// tryPresetClarify blocks on the interactive clarify UI when TurnPlan already
// carries a preset question/choices (ambiguous domain). Skips waiting for the
// main model to call clarify and avoids getting stuck behind the memory gate.
func (l *Loop) tryPresetClarify(
	ctx context.Context,
	session *runtime.Session,
	turnPlan cognition.TurnPlan,
	toolCtx tools.Context,
	records *[]runtime.StepRecord,
	schemas []llm.ToolSchema,
) (runtime.TurnResult, bool) {
	if turnPlan.Mode != cognition.ModeClarify {
		return runtime.TurnResult{}, false
	}
	question := strings.TrimSpace(turnPlan.ClarifyQuestion)
	if question == "" || toolCtx.ClarifyFn == nil {
		return runtime.TurnResult{}, false
	}
	choices := append([]string(nil), turnPlan.ClarifyChoices...)
	l.emitStatus("clarify", question)
	l.emit("clarify_plan", map[string]any{
		"question": question,
		"choices":  choices,
	})
	l.recordInjectionStep(records, "clarify", "preset question: "+question)

	answer, ok := toolCtx.ClarifyFn(ctx, question, choices)
	answer = strings.TrimSpace(answer)
	if !ok || answer == "" {
		text := "已取消澄清。"
		if !ok && ctx.Err() != nil {
			text = "澄清已中断。"
		}
		session.AppendMessage(llm.Message{Role: llm.RoleAssistant, Content: text})
		return runtime.TurnResult{AssistantText: text, StepRecords: *records}, true
	}

	l.emit("clarify_answer", map[string]any{"answer": answer})
	l.recordInjectionStep(records, "clarify", "answer="+answer)
	child := l.RunTurn(ctx, session, answer, toolCtx, schemas)
	child.StepRecords = append(*records, child.StepRecords...)
	return child, true
}
