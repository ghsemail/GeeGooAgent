package retrievalgate_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
)

func TestShouldRetrieveEmptyMessage(t *testing.T) {
	got := retrievalgate.ShouldRetrieve(nil, nil, nil, "  ")
	if got.Retrieve {
		t.Fatal("expected skip for empty")
	}
}

func TestShouldRetrieveAlwaysWithoutLLM(t *testing.T) {
	prov := &countingProvider{body: `{"retrieve":false,"query":"","reason":"should not run"}`}
	got := retrievalgate.ShouldRetrieve(context.Background(), prov, nil, "MACD")
	if !got.Retrieve || got.Query != "MACD" {
		t.Fatalf("expected always retrieve, got %+v", got)
	}
	if prov.calls != 0 {
		t.Fatalf("LLM should not run, calls=%d", prov.calls)
	}
}

func TestHasMemoryCue(t *testing.T) {
	if retrievalgate.HasMemoryCue("MACD") {
		t.Fatal("MACD is not a memory cue")
	}
	if !retrievalgate.HasMemoryCue("还记得上次") {
		t.Fatal("expected memory cue")
	}
}

type countingProvider struct {
	calls int
	body  string
}

func (c *countingProvider) Model() string { return "count" }

func (c *countingProvider) Chat(context.Context, []llm.Message, []llm.ToolSchema, float64, int) (*llm.Response, error) {
	c.calls++
	return &llm.Response{Content: c.body}, nil
}
