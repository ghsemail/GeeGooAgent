package tools

import (
	"context"
	"testing"
)

func TestHandleClarifyBypassesPanelFrequency(t *testing.T) {
	setBypassPanelFrequencyClarify(func(_ any, question string, choices []string) (string, bool) {
		if question == "请选择 K 线周期" && len(choices) > 0 {
			return "60m", true
		}
		return "", false
	})
	t.Cleanup(func() { setBypassPanelFrequencyClarify(nil) })

	called := false
	res := handleClarify(Context{
		RuntimeSession: struct{}{},
		ClarifyFn: func(context.Context, string, []string) (string, bool) {
			called = true
			return "", false
		},
	}, map[string]any{
		"question": "请选择 K 线周期",
		"choices":  []any{"60m", "daily", "5m", "15m"},
	})
	if called {
		t.Fatal("ClarifyFn should not run when panel frequency bypass applies")
	}
	if res.Status != StatusOK || res.Summary != "60m" {
		t.Fatalf("res=%+v", res)
	}
	if res.Data["panel_frequency_bypass"] != true {
		t.Fatalf("data=%v", res.Data)
	}
}

func TestHandleClarifyStillCallsFnWithoutPanelFrequency(t *testing.T) {
	setBypassPanelFrequencyClarify(func(_ any, _ string, _ []string) (string, bool) {
		return "", false
	})
	t.Cleanup(func() { setBypassPanelFrequencyClarify(nil) })

	res := handleClarify(Context{
		RuntimeSession: struct{}{},
		ClarifyFn: func(context.Context, string, []string) (string, bool) {
			return "daily", true
		},
	}, map[string]any{
		"question": "请选择 K 线周期",
		"choices":  []any{"60m", "daily"},
	})
	if res.Summary != "daily" {
		t.Fatalf("res=%+v", res)
	}
}
