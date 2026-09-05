package retrievalgate_test

import (
	"context"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
)

func TestShouldRetrieveEmptyMessage(t *testing.T) {
	got := retrievalgate.ShouldRetrieve(nil, nil, nil, "  ")
	if got.Retrieve {
		t.Fatal("expected skip for empty")
	}
}

func TestShouldRetrieveSkipsLLMWithoutMemoryCue(t *testing.T) {
	prov := &countingProvider{body: `{"retrieve":true,"query":"MACD","reason":"should not run"}`}
	got := retrievalgate.ShouldRetrieve(context.Background(), prov, nil, "MACD")
	if got.Retrieve {
		t.Fatalf("expected skip, got %+v", got)
	}
	if prov.calls != 0 {
		t.Fatalf("LLM should not run without memory cue, calls=%d", prov.calls)
	}
}

func TestShouldRetrieveTimesOutFailClosed(t *testing.T) {
	start := time.Now()
	got := retrievalgate.ShouldRetrieve(context.Background(), hangProvider{}, nil, "还记得上次选的信号吗")
	if time.Since(start) > 4*time.Second {
		t.Fatal("gate timeout took too long")
	}
	if got.Retrieve {
		t.Fatalf("timeout should skip, got %+v", got)
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

type hangProvider struct{}

func (hangProvider) Model() string { return "hang" }

func (hangProvider) Chat(ctx context.Context, _ []llm.Message, _ []llm.ToolSchema, _ float64, _ int) (*llm.Response, error) {
	<-ctx.Done()
	return nil, ctx.Err()
}
