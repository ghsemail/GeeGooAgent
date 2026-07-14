package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestRebuildGatewaySyncsCatalogModelToConfig(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["model_id"] != "m2" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id": "m2", "name": "deepseek-v4", "token": "sk-x", "base_url": "https://api.example/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	useOps := true
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:       "openai",
				Model:          "gpt-5.5",
				TokenKey:       "local",
				CatalogModelID: "m2",
				UseOpsModel:    &useOps,
			},
			SignalBaseURL: srv.URL,
			Compression:   config.CompressionConfig{Enabled: boolPtr(false)},
		},
	}
	if err := application.RebuildGateway(); err != nil {
		t.Fatal(err)
	}
	if application.Config.LLM.Model != "deepseek-v4" {
		t.Fatalf("config model=%q want deepseek-v4", application.Config.LLM.Model)
	}
	if application.EffectiveLLMModel() != "deepseek-v4" {
		t.Fatalf("effective=%q", application.EffectiveLLMModel())
	}
}
