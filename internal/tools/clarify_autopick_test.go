package tools

import "testing"

func TestAutoClarifyChoiceFrequency(t *testing.T) {
	choices := []string{"5m", "60m", "daily"}
	got, ok := AutoClarifyChoice("请选择频率", choices)
	if !ok || got != "60m" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}

func TestAutoClarifyChoiceFirstFallback(t *testing.T) {
	choices := []string{"选项A", "选项B"}
	got, ok := AutoClarifyChoice("请确认", choices)
	if !ok || got != "选项A" {
		t.Fatalf("got %q ok=%v", got, ok)
	}
}
