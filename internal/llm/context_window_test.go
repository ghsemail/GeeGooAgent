package llm

import "testing"

func TestResolveContextWindowConfiguredWins(t *testing.T) {
	t.Parallel()
	if got := ResolveContextWindow("gpt-5.5", 64000); got != 64000 {
		t.Fatalf("got %d", got)
	}
}

func TestResolveContextWindowKnownModels(t *testing.T) {
	t.Parallel()
	cases := map[string]int{
		"gpt-5.5":           400_000,
		"deepseek-v4-flash": 128_000,
		"claude-sonnet-4":   200_000,
		"MiniMax-M2.1":      204_800,
		"unknown-xyz":       DefaultContextWindow,
	}
	for model, want := range cases {
		if got := ResolveContextWindow(model, 0); got != want {
			t.Fatalf("%s: got %d want %d", model, got, want)
		}
	}
}
