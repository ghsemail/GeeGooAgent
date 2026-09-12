package catalog

import (
	"strings"
	"testing"
)

func TestApplyStrategyBacktestDefaultsEmpty(t *testing.T) {
	body := map[string]any{}
	ApplyStrategyBacktestDefaults(body)
	if body["months_back"] != 3 {
		t.Fatalf("months_back=%v want 3", body["months_back"])
	}
	if body["period"] != "3m" {
		t.Fatalf("period=%v want 3m", body["period"])
	}
}

func TestApplyStrategyBacktestDefaultsCoercesString(t *testing.T) {
	body := map[string]any{"months_back": "1", "period": "1m"}
	ApplyStrategyBacktestDefaults(body)
	if body["months_back"] != 1 || body["period"] != "1m" {
		t.Fatalf("got months=%v period=%v", body["months_back"], body["period"])
	}
}

func TestApplyStrategyBacktestDefaultsKeepsExplicit(t *testing.T) {
	body := map[string]any{"months_back": 1, "period": "1m"}
	ApplyStrategyBacktestDefaults(body)
	if body["months_back"] != 1 || body["period"] != "1m" {
		t.Fatalf("got months=%v period=%v", body["months_back"], body["period"])
	}
}

func TestRunStrategyBacktestPeriodSchemaDefault(t *testing.T) {
	params := runStrategyBacktestParameters()
	props := params["properties"].(map[string]any)
	period := props["period"].(map[string]any)
	desc, _ := period["description"].(string)
	if !strings.Contains(desc, "默认 3m") {
		t.Fatalf("period description %q should say 默认 3m", desc)
	}
}
