package runtime

import (
	"strings"
	"testing"
)

func TestAgentEventEncodeLine(t *testing.T) {
	line, err := NewAgentEvent("turn_start", map[string]any{"user_text": "hi"}).EncodeLine()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(line), "\n") {
		t.Fatalf("expected newline suffix: %q", line)
	}
	if !strings.Contains(string(line), `"schema_version":1`) {
		t.Fatalf("missing schema version: %s", line)
	}
}
