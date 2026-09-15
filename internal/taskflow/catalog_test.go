package taskflow

import "testing"

func TestListTemplatesIncludesMultiStrategy(t *testing.T) {
	templates := ListTemplates()
	if len(templates) < 1 {
		t.Fatal("expected at least one template")
	}
	found := false
	for _, tmpl := range templates {
		if tmpl.ID == TemplateMultiStrategyCompare {
			found = true
			if tmpl.Status != "available" {
				t.Fatalf("status=%s want available", tmpl.Status)
			}
			if len(tmpl.Phases) < 3 {
				t.Fatalf("phases=%v", tmpl.Phases)
			}
		}
	}
	if !found {
		t.Fatal("multi_strategy_compare missing from catalog")
	}
}

func TestCardPayload(t *testing.T) {
	flow := &Flow{
		RunID: "flow-test", Template: TemplateMultiStrategyCompare,
		Status: StatusRunning, Phase: PhaseProbeForeach,
		StockCode: "00700.HK", StockName: "腾讯控股",
		Strategies: []StrategyItem{
			{Label: "Macd4H", Status: StepDone, BuyHits: 2, SellHits: 1},
			{Label: "共振", Status: StepRunning},
		},
	}
	card := CardPayload(flow)
	if card["run_id"] != "flow-test" {
		t.Fatalf("card=%v", card)
	}
	if card["done"] != 1 {
		t.Fatalf("done=%v want 1", card["done"])
	}
}
