package tools

import (
	"fmt"
	"sort"
	"strings"
)

// Toolset is a named tool group (Hermes toolset parity).
type Toolset struct {
	ID          string
	Label       string
	Description string
	// ChatDefault includes this set in the default chat allowlist.
	ChatDefault bool
	names       map[string]struct{}
}

// Names returns sorted tool names in this toolset.
func (t Toolset) Names() []string {
	out := make([]string, 0, len(t.names))
	for name := range t.names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Contains reports whether tool name belongs to the toolset.
func (t Toolset) Contains(name string) bool {
	_, ok := t.names[name]
	return ok
}

func newToolset(id, label, desc string, chatDefault bool, names map[string]struct{}) Toolset {
	cp := make(map[string]struct{}, len(names))
	for k := range names {
		cp[k] = struct{}{}
	}
	return Toolset{
		ID: id, Label: label, Description: desc,
		ChatDefault: chatDefault, names: cp,
	}
}

// legacyToolsetAliases maps deprecated toolset ids to their replacements.
var legacyToolsetAliases = map[string][]string{
	"bot_manager":     {"trading_bot", "hedge_bot"},
	"market":          {"market", "analyst_runtime"},
	"market_data":     {"market"},
	"info_search":     {"market"},
	"research":        {"analyst_runtime"},
	"prompt_template": {"analyst_runtime", "prompt_admin", "custom_signal"},
}

func expandToolsetAlias(id string) []string {
	if ids, ok := legacyToolsetAliases[id]; ok {
		return append([]string(nil), ids...)
	}
	return []string{id}
}

// builtinToolsets is the canonical catalog. Order is display order.
var builtinToolsets = []Toolset{
	newToolset("market", "行情与资金", "交易日、搜码、行情、持仓、新闻检索", true, marketTools),
	newToolset("analyst_runtime", "运行时分析", "Prompt 列表、MCP 分析、资金面", true, analystRuntimeTools),
	newToolset("prompt_admin", "模板运营", "单项/竞品/ETF Prompt 模板 CRUD（高级）", false, promptAdminTools),
	newToolset("custom_signal", "定制策略", "定制策略定义与 CRUD（Monday · :3210）", true, customSignalTools),
	newToolset("strategy", "策略与回测", "信号列表、网格/DCA 生成与回测", true, strategyTools),
	newToolset("trading_bot", "交易机器人", "DCA/GRID/SmartTrade 读写", true, tradingBotTools),
	newToolset("hedge_bot", "对冲机器人", "HDG 对冲机器人读写", true, hedgeBotTools),
	newToolset("reminder_manager", "提醒机器人", "DCA/GRID/Smart 提醒读写", true, reminderManagerTools),
	newToolset("report_query", "报告查询", "读已有报告、Bot 态度与运行日志", true, reportQueryTools),
	newToolset("report_write", "报告写入", "Chat 中补写/修改盘前盘中盘后报告", true, reportWriteTools),
	newToolset("report_workflow", "报告 Workflow", "盘前/盘后自动化（默认不进 chat）", false, reportWorkflowTools),
	newToolset("agent_meta", "Agent 元能力", "记忆、澄清、委派等横切能力", true, agentMetaTools),
}

// workflowExclusiveTools are in report_workflow but not shared with any other toolset.
var workflowExclusiveTools = buildWorkflowExclusiveTools()

func buildWorkflowExclusiveTools() map[string]struct{} {
	shared := map[string]struct{}{}
	for _, ts := range builtinToolsets {
		if ts.ID == "report_workflow" {
			continue
		}
		for name := range ts.names {
			if _, wf := reportWorkflowTools[name]; wf {
				shared[name] = struct{}{}
			}
		}
	}
	exclusive := make(map[string]struct{}, len(reportWorkflowTools))
	for name := range reportWorkflowTools {
		if _, ok := shared[name]; !ok {
			exclusive[name] = struct{}{}
		}
	}
	return exclusive
}

// chatExcludedTools are in ChatDefault toolsets but omitted from default interactive chat.
var chatExcludedTools = map[string]struct{}{
	"recall":            {},
	"add_custom_signal": {},
	"edit_custom_signal": {},
	"delete_custom_signal": {},
}

// AllToolsets returns the built-in toolset catalog.
func AllToolsets() []Toolset {
	out := make([]Toolset, len(builtinToolsets))
	copy(out, builtinToolsets)
	return out
}

// ToolsetByID looks up a toolset by id (case-insensitive).
// Legacy ids bot_manager / market return merged compatibility toolsets.
func ToolsetByID(id string) (Toolset, bool) {
	want := strings.ToLower(strings.TrimSpace(id))
	switch want {
	case "bot_manager":
		return mergedLegacyToolset("bot_manager", "交易机器人（兼容）", "DCA/GRID/SmartTrade/HDG 读写", true, botManagerTools), true
	}
	for _, ts := range builtinToolsets {
		if ts.ID == want {
			return ts, true
		}
	}
	return Toolset{}, false
}

func mergedLegacyToolset(id, label, desc string, chatDefault bool, names map[string]struct{}) Toolset {
	return newToolset(id, label, desc, chatDefault, names)
}

// DefaultChatToolsetIDs returns ids with ChatDefault=true.
func DefaultChatToolsetIDs() []string {
	var out []string
	for _, ts := range builtinToolsets {
		if ts.ChatDefault {
			out = append(out, ts.ID)
		}
	}
	return out
}

// NormalizeToolsetIDs validates and lowercases ids. Unknown ids return an error.
// Empty input means "use defaults".
func NormalizeToolsetIDs(ids []string) ([]string, error) {
	if len(ids) == 0 {
		return DefaultChatToolsetIDs(), nil
	}
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range ids {
		id := strings.ToLower(strings.TrimSpace(raw))
		if id == "" || id == "all" || id == "default" {
			continue
		}
		for _, expanded := range expandToolsetAlias(id) {
			if _, ok := ToolsetByID(expanded); !ok {
				return nil, fmt.Errorf("unknown toolset: %s (use /toolsets)", raw)
			}
			if _, dup := seen[expanded]; dup {
				continue
			}
			seen[expanded] = struct{}{}
			out = append(out, expanded)
		}
	}
	if len(out) == 0 {
		return DefaultChatToolsetIDs(), nil
	}
	return out, nil
}

// ChatToolNamesForToolsets returns the chat allowlist for the given toolset ids.
// Future: filter by per-user entitlements (mcp_token → tool/skill ACL).
func ChatToolNamesForToolsets(ids []string) []string {
	normalized, err := NormalizeToolsetIDs(ids)
	if err != nil {
		normalized = DefaultChatToolsetIDs()
	}
	chat := map[string]struct{}{}
	for _, id := range normalized {
		ts, ok := ToolsetByID(id)
		if !ok {
			continue
		}
		for name := range ts.names {
			if _, skip := chatExcludedTools[name]; skip {
				continue
			}
			if _, onlyWorkflow := workflowExclusiveTools[name]; onlyWorkflow && id != "report_workflow" {
				continue
			}
			chat[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(chat))
	for name := range chat {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// RegisteredChatToolNamesFor returns allowlisted tools present in the registry
// for the selected toolsets. Empty ids → default chat toolsets.
func RegisteredChatToolNamesFor(registry *Registry, toolsetIDs []string) []string {
	if registry == nil {
		return nil
	}
	registered := map[string]struct{}{}
	for _, name := range registry.ListNames() {
		registered[name] = struct{}{}
	}
	var out []string
	for _, name := range ChatToolNamesForToolsets(toolsetIDs) {
		if _, ok := registered[name]; ok {
			out = append(out, name)
		}
	}
	return out
}

// FormatToolsetsListing renders /toolsets help text.
func FormatToolsetsListing(active []string) string {
	activeSet := map[string]struct{}{}
	normalized, _ := NormalizeToolsetIDs(active)
	for _, id := range normalized {
		activeSet[id] = struct{}{}
	}
	var lines []string
	lines = append(lines, "可用 toolset（Hermes 风格工具分组）：")
	for _, ts := range builtinToolsets {
		mark := " "
		if _, ok := activeSet[ts.ID]; ok {
			mark = "*"
		}
		chat := "chat"
		if !ts.ChatDefault {
			chat = "workflow"
		}
		lines = append(lines, fmt.Sprintf("  %s %-18s [%s] %s (%d tools)",
			mark, ts.ID, chat, ts.Label, len(ts.names)))
	}
	lines = append(lines, "")
	lines = append(lines, "切换: /toolsets market,strategy  或  /toolsets default（market/bot_manager 兼容旧配置）")
	return strings.Join(lines, "\n")
}
