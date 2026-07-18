package tools

import "testing"

func TestNormalizeHTTPResponseLoopback(t *testing.T) {
	t.Parallel()
	payload := map[string]any{
		"code":        "01810.HK",
		"finalValue":  99919.0,
		"profit_rate": -0.081,
	}
	normalized, summary := normalizeHTTPResponse("loopback_strategy", payload)
	if normalized["finalValue"] != 99919.0 {
		t.Fatalf("unexpected normalized payload: %#v", normalized)
	}
	if summary == "" || summary == "loopback_strategy succeeded" {
		t.Fatalf("expected profit summary, got %q", summary)
	}
}
