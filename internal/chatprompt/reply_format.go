package chatprompt

// FinalReplyFormatReminder is a short, non-persistent user message injected before the
// model composes a post-tool final answer. Complements SOUL rules without bloating system.
const FinalReplyFormatReminder = `[FORMAT] 这是给用户看的最终答复，请严格遵守 SOUL 的 Markdown 规则：
- ## / ### 标题独占一行，标题前后留空行
- 列表每项以「- 」开头并换行；加粗用 **文本**；引用用「> 」
- 多字段对比用「**字段**：值」列表，不要用 |表格|，不要用 --- 分隔线
- 若引用了 get_mcp_analysis 的 analysis_result，请改写排版，勿照抄其中的表格或粘连一行`

// ReplyFormatReminder returns the reminder when the turn is synthesizing after tools.
func ReplyFormatReminder(toolRound int) string {
	if toolRound < 1 {
		return ""
	}
	return FinalReplyFormatReminder
}
