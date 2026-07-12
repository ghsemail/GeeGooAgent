package chatui

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
  /flow              最近事件总线记录（可加数量）
  /run pre_market    运行盘前 workflow
  /dry-run on        开启 dry-run（跳过写 API）
  /dry-run off       关闭 dry-run
  /model             列出或切换 LLM 模型
  /verbose on        显示思考与 Tool 过程
  /verbose off       隐藏思考与 Tool 过程
  /think on          开启 DeepSeek 思考模式（写入 config）
  /think off         关闭 DeepSeek 思考模式
  /think auto        恢复模型默认思考策略
`
}

// SlashCommands for go-prompt completion.
var SlashCommands = []struct {
	Command     string
	Description string
}{
	{"/help", "显示帮助"},
	{"/exit", "退出并保存会话"},
	{"/quit", "退出并保存会话"},
	{"/session", "当前会话 ID 与消息数"},
	{"/tools", "列出可用 Tool"},
	{"/toolsets", "列出或切换 toolset"},
	{"/trace", "最近执行步骤"},
	{"/flow", "最近事件总线记录"},
	{"/run pre_market", "运行盘前 workflow"},
	{"/dry-run on", "开启 dry-run"},
	{"/dry-run off", "关闭 dry-run"},
	{"/model", "列出或切换 LLM 模型"},
	{"/verbose on", "显示 Tool 过程"},
	{"/verbose off", "隐藏 Tool 过程"},
	{"/think on", "开启思考模式"},
	{"/think off", "关闭思考模式"},
	{"/think auto", "恢复默认思考策略"},
}
