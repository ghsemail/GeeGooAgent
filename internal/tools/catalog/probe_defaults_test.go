package catalog

import "testing"

func TestApplyProbeDefaultsEmpty(t *testing.T) {
	body := map[string]any{}
	ApplyProbeDefaults(body)
	if body["months_back"] != 3 {
		t.Fatalf("months_back=%v want 3", body["months_back"])
	}
}

func TestApplyProbeDefaultsCoercesString(t *testing.T) {
	body := map[string]any{"months_back": "1"}
	ApplyProbeDefaults(body)
	if body["months_back"] != 1 {
		t.Fatalf("months_back=%v want 1", body["months_back"])
	}
}

func TestApplyProbeDefaultsKeepsExplicitLimit(t *testing.T) {
	body := map[string]any{"limit": "200"}
	ApplyProbeDefaults(body)
	if body["limit"] != 200 {
		t.Fatalf("limit=%v want 200", body["limit"])
	}
	if _, ok := body["months_back"]; ok {
		t.Fatalf("should not set months_back when limit provided: %v", body)
	}
}
