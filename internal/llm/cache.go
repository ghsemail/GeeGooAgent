package llm

// ApplyCacheBreakpoints marks stable prefix boundaries on a message copy.
//
// Breakpoint 1: system prompt (soul + tools + memory — byte-stable across turns).
// Breakpoint 2: last message before the volatile user tail (injected context + latest question).
//
// Markers are consumed by OpenAI-compatible providers that support cache_control.
// DeepSeek also benefits from automatic prefix caching when the stable prefix is unchanged.
func ApplyCacheBreakpoints(messages []Message) []Message {
	if len(messages) == 0 {
		return messages
	}
	out := make([]Message, len(messages))
	copy(out, messages)

	if out[0].Role == RoleSystem {
		out[0].CacheBreakpoint = true
	}
	if idx := stablePrefixEnd(out); idx >= 0 && idx < len(out) {
		out[idx].CacheBreakpoint = true
	}
	return out
}

// stablePrefixEnd returns the index of the last stable message before the volatile
// trailing user block (e.g. injected Tool 活动 + latest user question).
func stablePrefixEnd(messages []Message) int {
	if len(messages) <= 1 {
		return -1
	}
	i := len(messages) - 1
	for i > 0 && messages[i].Role == RoleUser {
		i--
	}
	if i <= 0 && messages[0].Role == RoleSystem {
		return -1
	}
	return i
}

// ResolveExplicitPromptCache decides whether to emit cache_control on breakpoints.
func ResolveExplicitPromptCache(name ProviderName, override *bool) bool {
	if override != nil {
		return *override
	}
	switch name {
	case ProviderDeepSeek, ProviderMinimax:
		return true
	default:
		return false
	}
}
