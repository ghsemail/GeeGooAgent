package llm

import "strings"

// chatCompletionsURL builds the POST URL for OpenAI-compatible chat APIs.
// Catalog rows may store host-only base_url (e.g. https://api.minimaxi.com) or a /v1 prefix.
func chatCompletionsURL(baseURL string) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "https://api.openai.com/v1/chat/completions"
	}
	if strings.HasSuffix(base, "/chat/completions") {
		return base
	}
	if strings.HasSuffix(base, "/v1") {
		return base + "/chat/completions"
	}
	return base + "/v1/chat/completions"
}
