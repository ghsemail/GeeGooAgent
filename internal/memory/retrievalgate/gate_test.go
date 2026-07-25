package retrievalgate_test

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory/retrievalgate"
)

func TestShouldRetrieveEmptyMessage(t *testing.T) {
	got := retrievalgate.ShouldRetrieve(nil, nil, nil, "  ")
	if got.Retrieve {
		t.Fatal("expected skip for empty")
	}
}
