package fts_test

import "testing"

import "github.com/ghsemail/GeeGooAgent/internal/memory/fts"

func TestBuildQuery(t *testing.T) {
	if got := fts.BuildQuery("Hello World"); got != "hello | world" {
		t.Fatalf("got %q", got)
	}
}
