package chatrepl_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/cli/chatrepl"
	"github.com/ghsemail/GeeGooAgent/internal/cli/progress"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func TestNDJSONEmitsTurnComplete(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "hello from agent"}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	application := &app.App{
		Config:   &config.AppConfig{OutputDir: t.TempDir()},
		Registry: registry,
		Gateway:  gateway,
		Agent:    agent.New(gateway, runtime.NewExecutor(registry), registry),
		State:    infra.NewStateStore(filepath.Join(t.TempDir(), "state")),
	}
	repl, err := chatrepl.New(application, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	sink := progress.NewNDJSONSink(&buf)
	if code := repl.RunSingleWithSink("ping", sink); code != 0 {
		t.Fatalf("exit=%d", code)
	}
	foundStart, foundComplete := false, false
	var assistantText string
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		var evt struct {
			SchemaVersion int            `json:"schema_version"`
			Event         string         `json:"event"`
			Data          map[string]any `json:"data"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &evt); err != nil {
			t.Fatalf("line: %v", err)
		}
		if evt.SchemaVersion != 1 {
			t.Fatalf("schema_version=%d", evt.SchemaVersion)
		}
		switch evt.Event {
		case "turn_start":
			foundStart = true
		case "turn_complete":
			foundComplete = true
			if s, _ := evt.Data["assistant_text"].(string); s != "" {
				assistantText = s
			}
		}
	}
	if !foundStart || !foundComplete {
		t.Fatalf("start=%v complete=%v body=%s", foundStart, foundComplete, buf.String())
	}
	if !strings.Contains(assistantText, "hello from agent") {
		t.Fatalf("assistant_text=%q body=%s", assistantText, buf.String())
	}
}

func TestNDJSONSinkSchemaVersion(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	sink := progress.NewNDJSONSink(&buf)
	sink.EmitProgress("tool_start", map[string]any{"name": "search_code"})
	var evt map[string]any
	if err := json.Unmarshal(buf.Bytes(), &evt); err != nil {
		t.Fatal(err)
	}
	if evt["schema_version"] != float64(1) {
		t.Fatalf("schema_version=%v", evt["schema_version"])
	}
	if evt["event"] != "tool_start" {
		t.Fatalf("event=%v", evt["event"])
	}
}

func TestAgentProgressMapsToNDJSON(t *testing.T) {
	t.Parallel()
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{Responses: []*llm.Response{{Content: "x"}}}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	loop := agent.NewLoop(gateway, runtime.NewExecutor(registry))
	var buf bytes.Buffer
	sink := progress.NewNDJSONSink(&buf)
	loop.SetProgress(sink.EmitProgress)
	_ = loop.RunTurn(context.Background(), runtime.NewSession(), "hi", tools.Context{}, nil)
	if !strings.Contains(buf.String(), `"event":"turn_start"`) {
		t.Fatalf("body=%s", buf.String())
	}
}
