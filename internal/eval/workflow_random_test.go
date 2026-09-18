package eval

import (
	"encoding/json"
	"testing"
)

func TestStrategyNamesFromCatalogPayload(t *testing.T) {
	raw, err := json.Marshal([]map[string]any{
		{"name": "Macd4H"},
		{"name": "SAR+MACD"},
	})
	if err != nil {
		t.Fatal(err)
	}
	names, err := strategyNamesFromCatalogPayload(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 2 || names[0] != "Macd4H" {
		t.Fatalf("names=%v", names)
	}

	wrapped, err := json.Marshal(map[string]any{
		"data": []map[string]any{{"name": "共振"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	names, err = strategyNamesFromCatalogPayload(wrapped)
	if err != nil {
		t.Fatal(err)
	}
	if len(names) != 1 || names[0] != "共振" {
		t.Fatalf("wrapped names=%v", names)
	}
}

func TestIndividualWorkflowEvalCasesIncludesRandom(t *testing.T) {
	var random *TurnPlanEvalCaseDef
	for _, c := range IndividualWorkflowEvalCases() {
		if c.ID == "workflow_generate_strategy_cognition_random" {
			random = &c
			break
		}
	}
	if random == nil {
		t.Fatal("missing random workflow case")
	}
	if !random.Options.RandomStrategyEnabled {
		t.Fatal("random case should enable random_strategy_enabled")
	}
}
