package runtimeapi

import (
	"encoding/json"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/runtime/events"
)

func TestClarifyProgressPayloadNeverNullChoices(t *testing.T) {
	raw, err := json.Marshal(ClarifyProgressPayload("s1", "选一个", nil))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if got["item_type"] != events.ItemClarifyPrompt {
		t.Fatalf("item_type=%v", got["item_type"])
	}
	if got["question"] != "选一个" {
		t.Fatalf("question=%v", got["question"])
	}
	choices, ok := got["choices"].([]any)
	if !ok {
		t.Fatalf("choices type %T (%s)", got["choices"], raw)
	}
	if choices == nil || len(choices) != 0 {
		t.Fatalf("expected empty JSON array, got %#v", got["choices"])
	}
	nested, _ := got["data"].(map[string]any)
	if nested == nil {
		t.Fatal("missing nested data")
	}
	if _, ok := nested["choices"].([]any); !ok {
		t.Fatalf("nested choices=%T", nested["choices"])
	}
}

func TestClarifyProgressPayloadLabeledChoices(t *testing.T) {
	got := ClarifyProgressPayload("s1", "找到多个标的，请选择：", []string{"00700 腾讯", "0700.HK 腾讯控股"})
	choices, _ := got["choices"].([]string)
	if len(choices) != 2 {
		t.Fatalf("choices=%v", got["choices"])
	}
	cards, ok := got["display_choices"].([]ClarifyChoiceCard)
	if !ok || len(cards) != 3 {
		t.Fatalf("display=%v", got["display_choices"])
	}
	if cards[0].Label != "A" || cards[2].Text == "" {
		t.Fatalf("cards=%v", cards)
	}
}
