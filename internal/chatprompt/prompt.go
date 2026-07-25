package chatprompt

// System returns the default geegoo chat system prompt.
func System() string {
	return DefaultBuilder().Build()
}

// SystemForUser builds the system prompt with tenant-scoped SOUL.
func SystemForUser(userID string) string {
	return SystemBuilder{Sections: []string{
		SoulForUser(userID),
		ToolRouting(),
		MemoryRules(),
		ServiceEndpoints(),
	}}.Build()
}
