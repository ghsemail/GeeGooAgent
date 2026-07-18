package flowview

import (
	"fmt"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/infra"
)

// Format renders one event bus record for /flow display.
func Format(rec infra.EventRecord) string {
	switch rec.Event {
	case "TurnStarted":
		return fmt.Sprintf("对话开始  session=%s", str(rec.Payload, "session_id"))
	case "TurnCompleted":
		return fmt.Sprintf("对话完成  session=%s  steps=%v", str(rec.Payload, "session_id"), rec.Payload["steps"])
	case "TurnFailed":
		return fmt.Sprintf("对话失败  session=%s  err=%s", str(rec.Payload, "session_id"), str(rec.Payload, "error"))
	case "TurnBudgetExhausted":
		return fmt.Sprintf("对话预算耗尽  session=%s  max_rounds=%v", str(rec.Payload, "session_id"), rec.Payload["max_rounds"])
	case "thinking_start":
		return "模型开始思考"
	case "thinking_stop":
		return "模型结束思考"
	case "step_complete":
		return fmt.Sprintf("回合完成  step=%v  round=%v  tools=%v",
			rec.Payload["step"], rec.Payload["round"], rec.Payload["had_tools"])
	case "RunStarted":
		return fmt.Sprintf("Skill 启动  skill=%s  session=%s", str(rec.Payload, "skill"), str(rec.Payload, "session_id"))
	case "RunCompleted":
		return fmt.Sprintf("Skill 完成  skill=%s  session=%s  status=%s  verdict=%s",
			str(rec.Payload, "skill"), str(rec.Payload, "session_id"),
			str(rec.Payload, "status"), str(rec.Payload, "verdict"))
	case "RunFailed":
		return fmt.Sprintf("Skill 失败  skill=%s  session=%s  err=%s",
			str(rec.Payload, "skill"), str(rec.Payload, "session_id"), str(rec.Payload, "error"))
	case "ToolCalled":
		return fmt.Sprintf("Tool 调用  %s  step=%v", str(rec.Payload, "tool"), rec.Payload["step"])
	case "ToolFinished":
		return fmt.Sprintf("Tool 结束  %s  step=%v", str(rec.Payload, "tool"), rec.Payload["step"])
	case "SynthesisStarted":
		return fmt.Sprintf("报告合成开始  %s (%s)  evidence=%v",
			str(rec.Payload, "code"), str(rec.Payload, "stock_name"), rec.Payload["evidence_count"])
	case "SynthesisCompleted":
		return fmt.Sprintf("报告合成完成  %s  suggestion=%s",
			str(rec.Payload, "code"), str(rec.Payload, "suggestion"))
	case "SynthesisFailed":
		return fmt.Sprintf("报告合成失败  %s  err=%s", str(rec.Payload, "code"), str(rec.Payload, "error"))
	case "SupervisorVerified":
		return fmt.Sprintf("Supervisor  verdict=%s  session=%s",
			str(rec.Payload, "verdict"), str(rec.Payload, "session_id"))
	default:
		return formatGeneric(rec)
	}
}

func formatGeneric(rec infra.EventRecord) string {
	parts := make([]string, 0, len(rec.Payload))
	for k, v := range rec.Payload {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, ", ")
}

func str(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
