package runtimeapi_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestClarifyHTTPE2E(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	client := mcp.NewClient("http://127.0.0.1:3120", "sk-test", mcp.Options{AllowedHosts: []string{"127.0.0.1"}})
	tools.RegisterBespokeTools(registry, tools.Deps{
		HTTP: tools.TestHTTPBackends(client), WorkspaceRoot: t.TempDir(),
	})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{
				ToolCalls: []llm.ToolCall{{
					ID: "c1", Name: "clarify",
					Arguments: map[string]any{
						"question": "pick market",
						"choices":  []any{"A-share", "HK"},
					},
				}},
			},
			{Content: "you picked A-share"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	application := &app.App{
		Config:   &config.AppConfig{},
		Registry: registry,
		Gateway:  gateway,
		Agent:    agent.New(gateway, runtime.NewExecutor(registry), registry),
		EventBus: infra.NewEventBus(),
	}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)

	const sessionID = "e2e-clarify"
	var sawClarify bool
	var wg sync.WaitGroup
	wg.Add(1)

	srv := httptest.NewServer(mux)
	defer srv.Close()

	go func() {
		defer wg.Done()
		payload := map[string]any{
			"model":  "geegoo-agent",
			"stream": true,
			"messages": []map[string]string{
				{"role": "user", "content": "analyze please"},
			},
		}
		body, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/completions", bytes.NewReader(body))
		if err != nil {
			t.Errorf("request: %v", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Errorf("post chat: %v", err)
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("chat status=%d", resp.StatusCode)
			return
		}
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			raw := strings.TrimPrefix(line, "data: ")
			if raw == "[DONE]" {
				break
			}
			var envelope map[string]json.RawMessage
			if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
				continue
			}
			if obj, ok := envelope["object"]; ok && string(obj) == `"geegoo.agent_event"` {
				var data struct {
					Event     string   `json:"event"`
					SessionID string   `json:"session_id"`
					Question  string   `json:"question"`
					Choices   []string `json:"choices"`
				}
				_ = json.Unmarshal(envelope["data"], &data)
				if data.Event == "clarify" {
					sawClarify = true
					clarifyBody, _ := json.Marshal(map[string]any{
						"session_id": sessionID,
						"answer":     "A-share",
					})
					clarifyBody, _ = json.Marshal(map[string]any{
						"session_id": data.SessionID,
						"answer":     "A-share",
					})
					clarifyReq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/chat/clarify", bytes.NewReader(clarifyBody))
					clarifyReq.Header.Set("Content-Type", "application/json")
					clarifyResp, err := http.DefaultClient.Do(clarifyReq)
					if err != nil {
						t.Errorf("clarify: %v", err)
						return
					}
					clarifyResp.Body.Close()
					if clarifyResp.StatusCode != http.StatusOK {
						t.Errorf("clarify status=%d", clarifyResp.StatusCode)
					}
				}
			}
		}
	}()

	waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-waitCtx.Done():
		t.Fatal("timeout waiting for stream chat")
	}
	if !sawClarify {
		t.Fatal("expected clarify agent_event in SSE stream")
	}
}

func TestClarifyHTTPNonStreamRequiresCallback(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	tools.RegisterBespokeTools(registry, tools.Deps{WorkspaceRoot: t.TempDir()})
	provider := &llm.MockProvider{
		Responses: []*llm.Response{
			{Content: "ok"},
		},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	application := &app.App{
		Config: &config.AppConfig{}, Registry: registry, Gateway: gateway,
		Agent: agent.New(gateway, runtime.NewExecutor(registry), registry),
	}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	srv := httptest.NewServer(mux)
	defer srv.Close()

	payload := map[string]any{
		"model": "geegoo-agent",
		"messages": []map[string]string{{"role": "user", "content": "hi"}},
	}
	body, _ := json.Marshal(payload)
	resp, err := http.Post(srv.URL+"/v1/chat/completions", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d body=%s", resp.StatusCode, raw)
	}
}
