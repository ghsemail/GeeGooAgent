package search_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/search"
)

func TestDuckDuckGoReturnsHits(t *testing.T) {
	t.Parallel()
	if os.Getenv("GEEGOO_RUN_NETWORK_TESTS") != "1" {
		t.Skip("set GEEGOO_RUN_NETWORK_TESTS=1 to run live DuckDuckGo integration test")
	}
	hits, err := search.DuckDuckGo(context.Background(), "SpaceX IPO 2024", 3)
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}
	if len(hits) == 0 {
		t.Skip("duckduckgo returned no results (network/rate limit)")
	}
	combined := strings.ToLower(hits[0].Title + hits[0].Snippet)
	if !strings.Contains(combined, "spacex") && !strings.Contains(combined, "space") {
		t.Logf("first hit: %+v", hits[0])
	}
}
