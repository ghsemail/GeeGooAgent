package runtimeapi

import "testing"

func TestPendingClarifyPayloadIncludesRecommendation(t *testing.T) {
	payload := pendingClarifyPayload(PendingClarify{
		SessionID:         "chat-1",
		Question:          "选哪个？",
		Choices:           []string{"A", "B"},
		RecommendedIndex:  1,
		RecommendedReason: "与上下文一致",
		AutoPickSeconds:   20,
	})
	if payload["recommended_index"] != 1 {
		t.Fatalf("recommended_index=%v", payload["recommended_index"])
	}
	if payload["recommended_choice"] != "B" {
		t.Fatalf("recommended_choice=%v", payload["recommended_choice"])
	}
	if payload["auto_pick_seconds"] != 20 {
		t.Fatalf("auto_pick_seconds=%v", payload["auto_pick_seconds"])
	}
}
