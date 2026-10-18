//go:build integration

package signalclient_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/lab/signalclient"
)

// Live integration against a running GeeGooSignal lab-api (feat/lab-api).
// Run: GEEGOO_AGENT_LAB_API_URL=http://host:3250 GEEGOO_SIGNAL_LAB_API_KEY=... go test -tags=integration ./internal/lab/signalclient/ -run Live -v
func TestLiveSignalLabAPIHealthAndKlineJob(t *testing.T) {
	base := strings.TrimSpace(os.Getenv("GEEGOO_AGENT_LAB_API_URL"))
	key := strings.TrimSpace(os.Getenv("GEEGOO_SIGNAL_LAB_API_KEY"))
	if base == "" || key == "" {
		t.Skip("set GEEGOO_AGENT_LAB_API_URL and GEEGOO_SIGNAL_LAB_API_KEY for live Signal lab-api")
	}
	client := signalclient.NewHTTPClient(base, key)
	client.PollBudget = 6 * time.Minute
	ctx := context.Background()
	if err := client.Health(ctx); err != nil {
		t.Fatalf("health: %v", err)
	}
	bars, err := client.FetchBars(ctx, signalclient.KlineJobRequest{
		Code: "00700.HK", Frequency: "daily", MonthsBack: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(bars) == 0 {
		t.Fatal("expected bars from live lab-api")
	}
}
