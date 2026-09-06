package agent_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
)

func TestRetrievalGateAlwaysRetrievesUtterance(t *testing.T) {
	got := retrievalgate.ShouldRetrieve(nil, nil, nil, "你好")
	if !got.Retrieve || got.Query != "你好" {
		t.Fatalf("greeting should retrieve: %+v", got)
	}
	got = retrievalgate.ShouldRetrieve(nil, nil, nil, "上次我们聊过什么")
	if !got.Retrieve {
		t.Fatalf("history cue should retrieve: %+v", got)
	}
}
