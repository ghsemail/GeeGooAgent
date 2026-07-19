package memport_test

import (
	"context"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
)

func TestNoopMemoryPassthrough(t *testing.T) {
	t.Parallel()
	m := memport.Noop()
	msgs := []llm.Message{{Role: llm.RoleUser, Content: "hi"}}
	out, err := m.Compress(context.Background(), memport.CompressInput{Messages: msgs})
	if err != nil {
		t.Fatal(err)
	}
	if out.DidCompress || len(out.Messages) != 1 {
		t.Fatalf("got %+v", out)
	}
}
