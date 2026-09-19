package premarket

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// Live e2e: SIGNAL_API_KEY required; optional SIGNAL_API_URL (default 146.56.225.252:3200).
func TestLivePremarketKeyLevelsChain(t *testing.T) {
	key := strings.TrimSpace(os.Getenv("SIGNAL_API_KEY"))
	if key == "" {
		t.Skip("SIGNAL_API_KEY not set")
	}
	url := strings.TrimSpace(os.Getenv("SIGNAL_API_URL"))
	if url == "" {
		url = "http://146.56.225.252:3200"
	}
	code := strings.TrimSpace(os.Getenv("KEYLEVEL_TEST_CODE"))
	if code == "" {
		code = "00700.HK"
	}

	ctx := context.Background()
	client := mcp.NewClient(url, key, mcp.Options{
		AllowedHosts: []string{"146.56.225.252", "127.0.0.1", "localhost"},
	})
	fetcher := SignalKeyLevelFetcher{Signal: client}

	levels, err := fetcher.FetchKeyLevels(ctx, code)
	if err != nil {
		t.Fatalf("FetchKeyLevels: %v", err)
	}
	if levels.Support == nil && levels.Resistance == nil {
		t.Fatalf("empty levels: %+v", levels)
	}

	resp, err := client.GetSupportingPrice(ctx, code, map[string]any{
		"include_60m": true, "include_legacy": true,
	})
	if err != nil {
		t.Fatalf("GetSupportingPrice: %v", err)
	}
	wire := map[string]any{
		"code": code, "current_price": resp.Data.CurrentPrice,
		"judgment": resp.Data.Judgment, "refs": resp.Data.Refs,
	}
	if resp.Data.Legacy != nil {
		wire["legacy"] = resp.Data.Legacy
	}
	b, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		t.Fatal(err)
	}

	store := memory.NewWorkingStore(infra.NewStateStore(t.TempDir()))
	w, err := store.Create("live-kl", "premarket_stock")
	if err != nil {
		t.Fatal(err)
	}
	w.Stocks[code] = memory.StockWorkspace{Code: code, Status: "pending"}
	updated, err := store.Apply(w, "get_key_levels", tools.Result{Status: tools.StatusOK, Data: data})
	if err != nil {
		t.Fatal(err)
	}
	ws := updated.Stocks[code]
	if !ws.KeyLevelsEngineOK {
		t.Fatal("KeyLevelsEngineOK false after get_key_levels apply")
	}
	if ws.KeyLevelSupportCenter <= 0 || ws.KeyLevelResistanceCenter <= 0 {
		t.Fatalf("centers not set: %+v", ws)
	}

	cached := keyLevelsFromWorkspace(ws)
	if !cached.Provenance.EngineUsed {
		t.Fatal("keyLevelsFromWorkspace should mark engine used")
	}
	merged := extractKeyLevels(ctx, ws)
	section := keyLevelEngineSection(ws, merged)
	if !strings.Contains(section, "支撑") || !strings.Contains(section, "阻力") {
		t.Fatalf("report section: %s", section)
	}
	t.Logf("OK %s support=%.2f resistance=%.2f section_len=%d",
		code, ws.KeyLevelSupportCenter, ws.KeyLevelResistanceCenter, len(section))
}
