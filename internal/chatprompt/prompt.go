package chatprompt

// System returns the default geegoo chat system prompt.
func System() string {
	return DefaultBuilder().Build()
}
