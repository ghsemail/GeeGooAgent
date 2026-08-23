package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/config"
)

func TestResolveLLMFromConfigOpsDefault(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["type"] != "configured" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id": "ops1", "name": "MiniMax-M3", "token": "sk-ops", "base_url": "https://api.minimax.io/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	useOps := true
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:    "openai",
				Model:       "deepseek-v4-pro",
				TokenKey:    "local",
				UseOpsModel: &useOps,
			},
			SignalBaseURL: srv.URL,
		},
	}
	resolved, err := application.resolveLLMFromConfig(application.Config.LLM)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.source != llmSourceOps || resolved.model != "MiniMax-M3" {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestResolveLLMFromConfigLocalExplicit(t *testing.T) {
	useOps := false
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:    "openai",
				Model:       "deepseek-v4-pro",
				TokenKey:    "local",
				UseOpsModel: &useOps,
			},
		},
	}
	resolved, err := application.resolveLLMFromConfig(application.Config.LLM)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.source != llmSourceLocal || resolved.model != "deepseek-v4-pro" {
		t.Fatalf("resolved=%+v", resolved)
	}
}

func TestResolveLLMFromConfigCatalogFailsNoFallback(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/getModel", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]map[string]any{})
	})
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"code":500,"message":"invalid ObjectID"}`, http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	useOps := false
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:       "openai",
				Model:          "deepseek-v4-pro",
				TokenKey:       "local",
				CatalogModelID: "bad-id",
				UseOpsModel:    &useOps,
			},
			SignalBaseURL: srv.URL,
		},
	}
	if _, err := application.resolveLLMFromConfig(application.Config.LLM); err == nil {
		t.Fatal("expected catalog resolve error")
	}
}

func TestRebuildGatewayUsesOpsWhenUseOpsModelTrue(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/queryModel", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		_ = json.NewDecoder(r.Body).Decode(&req)
		if req["type"] != "configured" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model_id": "ops1", "name": "MiniMax-M3", "token": "sk-ops", "base_url": "https://api.minimax.io/v1",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	useOps := true
	application := &App{
		Config: &config.AppConfig{
			LLM: config.LLMConfig{
				Provider:    "openai",
				Model:       "deepseek-v4-pro",
				TokenKey:    "local",
				UseOpsModel: &useOps,
			},
			SignalBaseURL: srv.URL,
			Compression:   config.CompressionConfig{Enabled: boolPtr(false)},
		},
	}
	if err := application.RebuildGateway(); err != nil {
		t.Fatal(err)
	}
	if application.EffectiveLLMModel() != "MiniMax-M3" {
		t.Fatalf("effective=%q want MiniMax-M3", application.EffectiveLLMModel())
	}
}
