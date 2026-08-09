package feishu

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildMarkdownPostRowsPlain(t *testing.T) {
	rows := BuildMarkdownPostRows("hello **world**")
	if len(rows) != 1 || len(rows[0]) != 1 {
		t.Fatalf("rows=%v", rows)
	}
	if rows[0][0]["tag"] != "md" || rows[0][0]["text"] != "hello **world**" {
		t.Fatalf("elem=%v", rows[0][0])
	}
}

func TestBuildMarkdownPostRowsSplitsFencedCode(t *testing.T) {
	src := "before\n```go\nfmt.Println(1)\n```\nafter"
	rows := BuildMarkdownPostRows(src)
	if len(rows) != 3 {
		t.Fatalf("want 3 rows, got %d: %#v", len(rows), rows)
	}
	if !strings.Contains(rows[1][0]["text"], "```go") {
		t.Fatalf("code row=%v", rows[1])
	}
	if rows[0][0]["text"] != "before" || rows[2][0]["text"] != "after" {
		t.Fatalf("prose rows=%v %v", rows[0], rows[2])
	}
}

func TestBuildMarkdownPostPayloadJSON(t *testing.T) {
	raw := BuildMarkdownPostPayload("# title")
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	zh, ok := parsed["zh_cn"].(map[string]any)
	if !ok {
		t.Fatalf("missing zh_cn: %s", raw)
	}
	content, ok := zh["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("content=%v", zh["content"])
	}
}
