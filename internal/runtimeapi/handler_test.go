package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/auth"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestHealthEndpoint(t *testing.T) {
	mux := httpserver.NewMux("agent-runtime")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["service"] != "agent-runtime" {
		t.Fatalf("service=%q", body["service"])
	}
}

func TestChatCompletionsWithMockLLM(t *testing.T) {
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "腾讯控股代码 00700.HK。", Usage: llm.TokenUsage{Model: "mock"}},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})

	application := &app.App{
		Config:   &config.AppConfig{},
		Registry: registry,
		Gateway:  gateway,
		Loop:     runtime.NewReActLoop(gateway, runtime.NewExecutor(registry)),
	}

	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application).Register(mux)
	handler := auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)

	payload := map[string]any{
		"model": "geegoo-agent",
		"messages": []map[string]string{
			{"role": "user", "content": "查腾讯"},
		},
		"stream": false,
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(raw))
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MCP-Token", "user-mcp-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		t.Fatalf("empty response: %+v", resp)
	}
	if resp.Choices[0].Message.Content != "腾讯控股代码 00700.HK。" {
		t.Fatalf("content=%q", resp.Choices[0].Message.Content)
	}
}

func TestChatCompletionsUnauthorized(t *testing.T) {
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(&app.App{}).Register(mux)
	handler := auth.BearerAPIKey("secret")(mux)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader([]byte(`{}`)))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", rec.Code)
	}
}
