package gateway

import (
	"fmt"
	"strings"
)

// FormatProgressLine maps Agent progress events to a Feishu progress bubble line.
// Returns ok=false for events that should not update the bubble.
func FormatProgressLine(event string, data map[string]any) (line string, ok bool) {
	name, _ := data["name"].(string)
	if name == "" {
		if names, okNames := data["tool_names"].([]string); okNames && len(names) > 0 {
			name = strings.Join(names, ", ")
		}
	}
	if name == "" {
		if names, okAny := data["tool_names"].([]any); okAny && len(names) > 0 {
			parts := make([]string, 0, len(names))
			for _, n := range names {
				if s, ok := n.(string); ok && s != "" {
					parts = append(parts, s)
				}
			}
			name = strings.Join(parts, ", ")
		}
	}

	switch event {
	case "tool_start":
		if name == "" {
			return "", false
		}
		return "⏳ `" + name + "`", true
	case "tool_done", "tool_intercepted":
		if name == "" {
			return "", false
		}
		status, _ := data["status"].(string)
		summary, _ := data["summary"].(string)
		icon := "✅"
		if status == "error" || status == "skip" {
			icon = "❌"
			if status == "skip" {
				icon = "⏭"
			}
		}
		extra := ""
		if ms, ok := asInt64(data["duration_ms"]); ok && ms > 0 {
			extra = fmt.Sprintf(" (%dms)", ms)
		}
		line = icon + " `" + name + "`" + extra
		if summary != "" && status != "ok" && status != "" {
			sum := strings.TrimSpace(summary)
			if len([]rune(sum)) > 80 {
				sum = string([]rune(sum)[:80]) + "…"
			}
			line += " — " + sum
		}
		return line, true
	case "llm_tools":
		if name == "" {
			return "🔧 准备调用工具…", true
		}
		return "🔧 准备调用: `" + name + "`", true
	case "plan_proposed":
		return "📋 待确认写操作计划", true
	case "error":
		msg, _ := data["message"].(string)
		if msg == "" {
			msg = "错误"
		}
		if len([]rune(msg)) > 120 {
			msg = string([]rune(msg)[:120]) + "…"
		}
		return "⚠️ " + msg, true
	default:
		return "", false
	}
}

func asInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case float64:
		return int64(n), true
	default:
		return 0, false
	}
}

// RenderProgressMarkdown builds the editable progress bubble body.
func RenderProgressMarkdown(lines []string) string {
	if len(lines) == 0 {
		return "**处理中…**"
	}
	var b strings.Builder
	b.WriteString("**处理中**\n\n")
	for _, line := range lines {
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
}
