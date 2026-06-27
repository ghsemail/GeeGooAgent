package runtime

import "github.com/ghsemail/GeeGooAgent/internal/chatprompt"

// ChatSystemPrompt returns the default system prompt for chat/runtime.
func ChatSystemPrompt() string { return chatprompt.System() }
