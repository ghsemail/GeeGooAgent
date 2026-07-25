package agent_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
)

func TestRetrievalGateHeuristicFallback(t *testing.T) {
	got := retrievalgate.ShouldRetrieve(nil, nil, nil, "你好")
	if got.Retrieve {
		t.Fatalf("greeting should skip: %+v", got)
	}
	got = retrievalgate.ShouldRetrieve(nil, nil, nil, "上次我们聊过什么")
	if !got.Retrieve {
		t.Fatalf("history cue should retrieve: %+v", got)
	}
}
