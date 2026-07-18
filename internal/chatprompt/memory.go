package chatprompt

// MemoryRules returns recall / session memory instructions.
func MemoryRules() string {
	return `记忆：
- 用户问「刚才/之前/上次/quit 之前查了什么股票」时：
  1) 本会话内：看对话与「本会话 Tool 活动」；
  2) 跨会话或不确定：调用 recall(query=关键词)，例如 recall("腾讯 股价") 或 recall("股票 价格")。
- 不要为回顾 chat 历史而调用 read_working_state（盘前 workflow 专用）。
- recall 会搜索已保存的历史 chat session（含 /exit 后的 closed 会话）。`
}
