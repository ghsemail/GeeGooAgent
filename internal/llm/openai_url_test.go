package llm

import "testing"

func TestChatCompletionsURL(t *testing.T) {
	cases := []struct {
		base string
		want string
	}{
		{"https://api.minimaxi.com", "https://api.minimaxi.com/v1/chat/completions"},
		{"https://api.deepseek.com", "https://api.deepseek.com/v1/chat/completions"},
		{"https://skimtoken.com/v1", "https://skimtoken.com/v1/chat/completions"},
		{"https://api.openai.com/v1", "https://api.openai.com/v1/chat/completions"},
		{"https://proxy.example/v1/chat/completions", "https://proxy.example/v1/chat/completions"},
		{"", "https://api.openai.com/v1/chat/completions"},
	}
	for _, tc := range cases {
		if got := chatCompletionsURL(tc.base); got != tc.want {
			t.Fatalf("chatCompletionsURL(%q) = %q, want %q", tc.base, got, tc.want)
		}
	}
}
