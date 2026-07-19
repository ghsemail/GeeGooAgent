package chatui

import "strings"

// SlashCommand is one slash command entry for help and completion.
type SlashCommand struct {
	Command     string
	Description string
}

// BuildHelpText returns slash command help (Python chat_commands.py).
func BuildHelpText() string {
	return `geegoo chat — 与 GeeGoo Agent 对话，并查看 Tool / workflow 轨迹

对话：直接输入问题，例如「分析一下腾讯」

斜杠命令（输入 / 可自动补全）：
  /help              显示帮助
  /exit              退出并保存会话
  /quit              退出并保存会话
  /session           当前会话 ID 与消息数
  /tools             列出可用 Tool
  /toolsets          列出或切换 toolset 分组
  /trace             最近执行步骤（可加数量）
  /flow              最近事件总线（Run/Turn/Tool/Synthesis，可加数量）
  /run pre_market    运行盘前 workflow
  /dry-run on        开启 dry-run（跳过写 API）
  /dry-run off       关闭 dry-run
  /model             列出或切换模型管理中的模型（默认 trading_operation 运营配置）
  /verbose on        显示思考与 Tool 过程
  /verbose off       隐藏思考与 Tool 过程
  /think on          开启 DeepSeek 思考模式（写入 config）
  /think off         关闭 DeepSeek 思考模式
  /think auto        恢复模型默认思考策略
  /details           折叠模式（Hermes 风格：hidden|collapsed|expanded|cycle）
  /stream on         流式显示助手回复（逐字输出）
  /stream off        等完整回复后再渲染（默认）
  /reply markdown    Markdown 排版（glamour，默认）
  /reply plain       纯文本排版
  /sessions          TUI：切换/新建 live session（Ctrl+X）
  /mouse             TUI：鼠标 tracking（off=可选中文字 wheel=滚轮滚动）
`
}

// SlashCommands for go-prompt and TUI completion.
var SlashCommands = []SlashCommand{
	{"/help", "显示帮助"},
	{"/exit", "退出并保存会话"},
	{"/quit", "退出并保存会话"},
	{"/session", "当前会话 ID 与消息数"},
	{"/tools", "列出可用 Tool"},
	{"/toolsets", "列出或切换 toolset"},
	{"/trace", "最近执行步骤"},
	{"/flow", "最近事件总线（Run/Turn/Tool/Synthesis）"},
	{"/run pre_market", "运行盘前 workflow"},
	{"/dry-run on", "开启 dry-run"},
	{"/dry-run off", "关闭 dry-run"},
	{"/model", "列出或切换 LLM 模型"},
	{"/verbose on", "显示 Tool 过程"},
	{"/verbose off", "隐藏 Tool 过程"},
	{"/think on", "开启思考模式"},
	{"/think off", "关闭思考模式"},
	{"/think auto", "恢复默认思考策略"},
	{"/details", "折叠模式 details_mode"},
	{"/stream on", "流式显示助手回复"},
	{"/stream off", "完整回复后再显示"},
	{"/reply markdown", "Markdown 排版"},
	{"/reply plain", "纯文本排版"},
	{"/sessions", "TUI live session 切换"},
	{"/mouse", "TUI 鼠标 tracking"},
}

// SlashCommandStrings returns command strings for bubbles textinput suggestions.
func SlashCommandStrings() []string {
	out := make([]string, len(SlashCommands))
	for i, c := range SlashCommands {
		out[i] = c.Command
	}
	return out
}

// MatchSlashCommands returns commands whose text starts with prefix (e.g. "/h", "/dry").
func MatchSlashCommands(prefix string) []SlashCommand {
	prefix = strings.TrimLeft(prefix, " ")
	if !strings.HasPrefix(prefix, "/") {
		return nil
	}
	var out []SlashCommand
	for _, item := range SlashCommands {
		if strings.HasPrefix(item.Command, prefix) {
			out = append(out, item)
		}
	}
	return out
}
