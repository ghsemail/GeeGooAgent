package catalog_test

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

func TestHTTPToolCatalogHasStableSpecs(t *testing.T) {
	t.Parallel()

	names := map[string]bool{}
	paths := map[string]string{}

	for _, spec := range catalog.AllHTTP() {
		if spec.Name == "" || spec.Path == "" {
			t.Fatalf("incomplete HTTP spec: %+v", spec)
		}
		if spec.Name == "switch_bot" || spec.Path == "/switchBot" {
			t.Fatalf("legacy switchBot proxy must not be registered: %+v", spec)
		}
		if !strings.HasPrefix(spec.Path, "/") {
			t.Fatalf("%s path must be absolute, got %q", spec.Name, spec.Path)
		}
		if catalog.BespokeNames[spec.Name] {
			t.Fatalf("%s should not be registered as a generic HTTP tool and a bespoke tool", spec.Name)
		}
		if names[spec.Name] {
			t.Fatalf("duplicate HTTP tool name %q", spec.Name)
		}
		names[spec.Name] = true
		if owner, ok := paths[spec.Path]; ok {
			t.Fatalf("duplicate HTTP path %q used by %s and %s", spec.Path, owner, spec.Name)
		}
		paths[spec.Path] = spec.Name
	}
}

func TestHighFrequencyBespokeMCPPathsAreRegistered(t *testing.T) {
	t.Parallel()

	for name, path := range map[string]string{
		"check_trading_day":          "/checkTradingDay",
		"get_current_price":          "/getCurrentPrice",
		"get_report_bot_codes":       "/getReportBotCodes",
		"get_mcp_analysis":           "/getMCPAnalysis",
		"get_stock_daily_reports":    "/getStockDailyReports",
		"get_capital_flow":           "/getCapitalFlow",
		"get_capital_distribution":   "/getCapitalDistribution",
		"fetch_market_news":          "/getMarketNews",
		"fetch_stock_news":           "/getStockNews",
		"get_bot_yesterday_attitude": "/getBotYesterdayAttitude",
		"create_pre_market_report":   "/createPreMarketReport",
	} {
		if !catalog.BespokeNames[name] {
			t.Fatalf("%s should be registered as a bespoke tool", name)
		}
		if !strings.HasPrefix(path, "/") {
			t.Fatalf("%s path must be absolute, got %q", name, path)
		}
	}
}
