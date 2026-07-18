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

// builtinToolsets is the canonical catalog. Order is display order.
var builtinToolsets = []Toolset{
	newToolset("market", "行情与分析", "行情、新闻、检索与 MCP 分析", true, marketTools),
	newToolset("strategy", "策略生成与回测", "网格/DCA 策略生成与回测", true, strategyTools),
	newToolset("bot_manager", "交易 Bot", "DCA/GRID/SmartTrade/HDG 读写", true, botManagerTools),
	newToolset("reminder_manager", "提醒 Bot", "DCA/GRID/Smart 提醒读写", true, reminderManagerTools),
	newToolset("report_query", "报告查询", "读盘前/盘中/盘后报告", true, reportQueryTools),
	newToolset("report_workflow", "报告 Workflow", "盘前/盘后自动化写报告（默认不进 chat）", false, reportWorkflowTools),
	newToolset("prompt_template", "Prompt 模板", "竞品/ETF 分析模板 CRUD（高级，默认不进 chat）", false, promptTemplateTools),
}

// workflowExclusiveTools are in report_workflow but not shared with any other toolset.
// Shared tools (e.g. get_bot_yesterday_attitude in market) stay in default chat.
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

// AllToolsets returns the built-in toolset catalog.
func AllToolsets() []Toolset {
	out := make([]Toolset, len(builtinToolsets))
	copy(out, builtinToolsets)
	return out
}

// ToolsetByID looks up a toolset by id (case-insensitive).
func ToolsetByID(id string) (Toolset, bool) {
	want := strings.ToLower(strings.TrimSpace(id))
	for _, ts := range builtinToolsets {
		if ts.ID == want {
			return ts, true
		}
	}
	return Toolset{}, false
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
		if _, ok := ToolsetByID(id); !ok {
			return nil, fmt.Errorf("unknown toolset: %s (use /toolsets)", raw)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return DefaultChatToolsetIDs(), nil
	}
	return out, nil
}

// ChatToolNamesForToolsets returns the chat allowlist for the given toolset ids.
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
	lines = append(lines, "切换: /toolsets market,bot_manager  或  /toolsets default")
	return strings.Join(lines, "\n")
}
