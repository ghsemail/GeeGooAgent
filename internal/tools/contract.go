package tools

import (
	"fmt"
	"strings"
	"time"
)

// EnvelopeInspection lets HTTP-forwarding tools record business metadata and
// detect "soft failures": the GeeGoo API returns code=100 (success) but the
// payload is empty or missing expected fields. Without this, downstream steps
// operate on empty data silently.

// EmptyResultTools are tools whose empty-but-code=100 response should be
// treated as Skip with a data-gap note rather than OK. Listing tools and
// analysis tools belong here.
var EmptyResultTools = map[string]bool{
	"list_dca_bots":         true,
	"list_grid_bots":        true,
	"list_smart_trades":     true,
	"list_hdg_bots":         true,
	"list_dca_reminders":    true,
	"list_grid_reminders":   true,
	"list_smart_reminders":  true,
	"get_mcp_analysis":      true,
	"get_position":          true,
	"get_ticker":            true,
	"get_broker":            true,
	"get_capital_flow":           true,
	"get_capital_distribution":   true,
	"get_bot_yesterday_attitude": true,
	"generate_grid_strategy":     true,
	"generate_dca_strategy":      true,
}

// ClassifyHTTPPayload decides status/summary from a normalized payload for
// HTTP-forwarded tools. Returns (status, summary, isDataGap).
func ClassifyHTTPPayload(toolName string, normalized map[string]any, rawEnvelope map[string]any) (Status, string, bool) {
	if !EmptyResultTools[toolName] {
		return StatusOK, "", false
	}
	if isEmptyPayload(toolName, normalized) {
		note := emptyDataNoteForTool(toolName, normalized)
		return StatusSkip, note, true
	}
	return StatusOK, "", false
}

// isEmptyPayload reports whether the normalized result has no usable items.
func isEmptyPayload(toolName string, normalized map[string]any) bool {
	if normalized == nil {
		return true
	}
	if items, ok := normalized["items"].([]any); ok && len(items) == 0 {
		return true
	}
	if c, ok := normalized["count"].(int); ok && c == 0 {
		return true
	}
	// analysis_result empty string → treat as empty regardless of metadata
	// fields (model/create_date are not analysis content).
	if ar, ok := normalized["analysis_result"].(string); ok && strings.TrimSpace(ar) == "" {
		return true
	}
	switch toolName {
	case "generate_grid_strategy":
		return !gridStrategyHasUsableData(normalized)
	case "generate_dca_strategy":
		return !dcaStrategyHasUsableData(normalized)
	}
	return false
}

func gridStrategyHasUsableData(m map[string]any) bool {
	param, ok := m["param"].(map[string]any)
	if !ok || len(param) == 0 {
		return false
	}
	for _, key := range []string{"upper_limit_price", "lower_limit_price", "grid_num"} {
		if numericNonZero(param[key]) {
			return true
		}
	}
	return false
}

func dcaStrategyHasUsableData(m map[string]any) bool {
	if signal, ok := m["signal"].(map[string]any); ok {
		if buy, ok := signal["buy_signal"].([]any); ok && len(buy) > 0 {
			return true
		}
	}
	return false
}

func numericNonZero(v any) bool {
	switch n := v.(type) {
	case float64:
		return n != 0
	case float32:
		return n != 0
	case int:
		return n != 0
	case int64:
		return n != 0
	default:
		return false
	}
}

func emptyDataNoteForTool(toolName string, normalized map[string]any) string {
	code, _ := normalized["code"].(string)
	switch toolName {
	case "get_position":
		return fmt.Sprintf("get_position: 无持仓数据（富途未配置或空仓；code=%s）", code)
	case "get_ticker":
		return fmt.Sprintf("get_ticker: 无逐笔数据（富途 OpenD 未配置或非交易时段；code=%s）", code)
	case "get_broker":
		return fmt.Sprintf("get_broker: 无经纪分布（富途未配置；code=%s）", code)
	case "get_mcp_analysis":
		return fmt.Sprintf("get_mcp_analysis: 分析结果为空（analyze-api 与 mcp-api 均无内容；code=%s）", code)
	case "generate_grid_strategy":
		if code != "" {
			return fmt.Sprintf("generate_grid_strategy: 未返回可用网格 param（upper/lower/grid_num；code=%s）；检查 analyze-api :3230 与 LLM/prompt 配置", code)
		}
		return "generate_grid_strategy: 未返回可用网格 param（upper_limit_price / lower_limit_price / grid_num）；检查 analyze-api :3230 与 LLM/prompt 配置"
	case "generate_dca_strategy":
		if code != "" {
			return fmt.Sprintf("generate_dca_strategy: 未返回 signal.buy_signal（code=%s）；确认 signal_id 有效且 analyze-api 已配置", code)
		}
		return "generate_dca_strategy: 未返回 signal.buy_signal；确认 signal_id 有效且 analyze-api 已配置"
	default:
		if code != "" {
			return fmt.Sprintf("%s: API 返回成功但数据为空（code=%s）", toolName, code)
		}
		return fmt.Sprintf("%s: API 返回成功但数据为空（可能无此标的/无记录）", toolName)
	}
}

// MetaFromEnvelope extracts business metadata from a GeeGoo envelope for Result.Meta.
func MetaFromEnvelope(envelope map[string]any, started time.Time) map[string]any {
	meta := map[string]any{
		"duration_ms": time.Since(started).Milliseconds(),
	}
	if envelope != nil {
		if code, ok := envelope["code"]; ok {
			meta["api_code"] = code
		}
		if msg, ok := envelope["message"].(string); ok && msg != "" {
			meta["api_message"] = msg
		}
	}
	return meta
}
