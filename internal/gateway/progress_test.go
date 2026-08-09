package gateway

import (
	"strings"
	"testing"
)

func TestFormatProgressLineToolStartDone(t *testing.T) {
	line, ok := FormatProgressLine("tool_start", map[string]any{"name": "search_code"})
	if !ok || line != "⏳ `search_code`" {
		t.Fatalf("start=%q ok=%v", line, ok)
	}
	line, ok = FormatProgressLine("tool_done", map[string]any{
		"name": "search_code", "status": "ok", "duration_ms": int64(42),
	})
	if !ok || line != "✅ `search_code` (42ms)" {
		t.Fatalf("done=%q ok=%v", line, ok)
	}
}

func TestFormatProgressLineSkipsUnknown(t *testing.T) {
	if _, ok := FormatProgressLine("llm_delta", map[string]any{"text": "x"}); ok {
		t.Fatal("expected skip")
	}
}

func TestRenderProgressMarkdown(t *testing.T) {
	got := RenderProgressMarkdown([]string{"⏳ `a`", "✅ `a` (1ms)"})
	for _, part := range []string{"**处理中**", "⏳ `a`", "✅ `a` (1ms)"} {
		if !strings.Contains(got, part) {
			t.Fatalf("missing %q in %q", part, got)
		}
	}
}
