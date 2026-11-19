package app

import (
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestStrategyDevJevConfig_DataBearerWithoutOpsCatalog(t *testing.T) {
	t.Setenv("GEEGOO_DATA_USHK_TOKEN", "data-bearer-test")
	cfg := &config.AppConfig{
		DataNodes: []config.DataNodeConfig{
			{ID: "us-hk", Regions: []string{"HK"}, BaseURL: "http://data.example:3300", BearerEnv: "GEEGOO_DATA_USHK_TOKEN"},
		},
	}
	a := &App{Config: cfg}
	jc := a.StrategyDevJevConfig()
	if jc.APIKey != "data-bearer-test" {
		t.Fatalf("api_key=%q", jc.APIKey)
	}
	if jc.BaseURL != "http://data.example:3300" {
		t.Fatalf("base_url=%q", jc.BaseURL)
	}
	if jc.Model != config.DefaultDecisionModel {
		t.Fatalf("model=%q", jc.Model)
	}
}