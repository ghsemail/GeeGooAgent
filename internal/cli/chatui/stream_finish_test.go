package chatui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestFinishAssistantStreamPrefersNormalizedFinalText(t *testing.T) {
	on := true
	var buf bytes.Buffer
	ui := New(&buf)
	ui.setPlain(false)
	ui.width = 100
	ui.ApplyDisplay(config.DisplayConfig{StreamReply: &on})

	ui.EmitProgress("turn_start", nil)
	ui.EmitProgress("stream_delta", map[string]any{"content": "##你好粘在一行"})
	normalized := "## 你好\n\n- 列表项"
	if !ui.FinishAssistantStream(normalized) {
		t.Fatal("expected streamed final reply")
	}
	out := stripANSI(buf.String())
	if strings.Contains(out, "##你好") {
		t.Fatalf("should render normalized final text, not raw stream buffer: %q", out)
	}
	if !strings.Contains(out, "你好") || !strings.Contains(out, "列表项") {
		t.Fatalf("missing normalized content: %q", out)
	}
}
