package runtimeapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/agent"
	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/auth"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/httpserver"
	"github.com/ghsemail/GeeGooAgent/internal/infra"
	"github.com/ghsemail/GeeGooAgent/internal/llm"
	"github.com/ghsemail/GeeGooAgent/internal/runtime"
	"github.com/ghsemail/GeeGooAgent/internal/runtimeapi"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

func testEvalAutoHandler(t *testing.T) http.Handler {
	t.Helper()
	db, err := infra.OpenSQLite(filepath.Join(t.TempDir(), "geegoo.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	cfg := &config.AppConfig{UserMCPToken: "test-mcp-token"}
	registry := tools.NewRegistry()
	provider := &llm.MockProvider{
		Responses: []*llm.Response{{Content: "腾讯控股现价约 320 港元。", Usage: llm.TokenUsage{Model: "mock"}}},
	}
	gateway := llm.NewGateway(provider, llm.GatewayConfig{MaxRetries: 1})
	gateway.SetSleep(func(time.Duration) {})
	application := &app.App{
		Config:   cfg,
		Registry: registry,
		Gateway:  gateway,
		Agent:    agent.New(gateway, runtime.NewExecutor(registry), registry),
		DB:       db,
	}
	mux := httpserver.NewMux("agent-runtime")
	runtimeapi.NewHandler(application, "").Register(mux)
	return auth.SkipPaths(map[string]struct{}{"/health": {}}, auth.BearerAPIKey("test-runtime-key"))(mux)
}

func evalAutoJSON(t *testing.T, handler http.Handler, method, path string, body any) (int, map[string]any) {
	t.Helper()
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	out := map[string]any{}
	if rec.Body.Len() > 0 {
		_ = json.Unmarshal(rec.Body.Bytes(), &out)
	}
	return rec.Code, out
}

func TestEvalCatalogListsTurnPlanCategories(t *testing.T) {
	handler := testEvalAutoHandler(t)
	code, body := evalAutoJSON(t, handler, http.MethodGet, "/v1/dashboard/eval/catalog", nil)
	if code != http.StatusOK {
		t.Fatalf("status=%d body=%v", code, body)
	}
	cats, _ := body["categories"].([]any)
	if len(cats) < 8 {
		t.Fatalf("categories=%d want >=8 body=%v", len(cats), body)
	}
	cases, _ := body["cases"].([]any)
	if len(cases) < 24 {
		t.Fatalf("cases=%d want >=24", len(cases))
	}
	first, _ := cases[0].(map[string]any)
	if first["id"] == "" || first["category"] == "" || first["message"] == "" {
		t.Fatalf("case missing fields: %v", first)
	}
}

func TestEvalSuiteCreateAndList(t *testing.T) {
	handler := testEvalAutoHandler(t)
	code, body := evalAutoJSON(t, handler, http.MethodPost, "/v1/dashboard/eval/suites", map[string]any{
		"title":        "股票分析冒烟",
		"category_ids": []string{"stock_analysis"},
	})
	if code != http.StatusOK {
		t.Fatalf("create status=%d body=%v", code, body)
	}
	suite, _ := body["suite"].(map[string]any)
	id, _ := suite["id"].(string)
	if id == "" {
		t.Fatalf("missing suite id: %v", body)
	}
	caseIDs, _ := suite["case_ids"].([]any)
	if len(caseIDs) != 5 {
		t.Fatalf("stock_analysis cases=%d want 5", len(caseIDs))
	}

	code, body = evalAutoJSON(t, handler, http.MethodGet, "/v1/dashboard/eval/suites", nil)
	if code != http.StatusOK {
		t.Fatalf("list status=%d body=%v", code, body)
	}
	suites, _ := body["suites"].([]any)
	if len(suites) != 1 {
		t.Fatalf("suites=%d want 1", len(suites))
	}
}

func TestEvalJobCreateRunsSelectedCategory(t *testing.T) {
	handler := testEvalAutoHandler(t)
	code, body := evalAutoJSON(t, handler, http.MethodPost, "/v1/dashboard/eval/jobs", map[string]any{
		"title":        "单条冒烟",
		"category_ids": []string{"chat"},
		"case_ids":     []string{"turn_plan_chat_definition"},
	})
	if code != http.StatusOK {
		t.Fatalf("create job status=%d body=%v", code, body)
	}
	job, _ := body["job"].(map[string]any)
	id, _ := job["id"].(string)
	if id == "" {
		t.Fatalf("missing job id: %v", body)
	}
	items, _ := job["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}

	deadline := time.Now().Add(8 * time.Second)
	var got map[string]any
	for time.Now().Before(deadline) {
		code, got = evalAutoJSON(t, handler, http.MethodGet, "/v1/dashboard/eval/jobs/"+id, nil)
		if code != http.StatusOK {
			t.Fatalf("get job status=%d body=%v", code, got)
		}
		job, _ = got["job"].(map[string]any)
		status, _ := job["status"].(string)
		if status == "pass" || status == "fail" || status == "error" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	job, _ = got["job"].(map[string]any)
	status, _ := job["status"].(string)
	if status == "queued" || status == "running" {
		t.Fatalf("job did not finish: %v", got)
	}
	items, _ = got["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("items=%d", len(items))
	}
	item, _ := items[0].(map[string]any)
	sessionID, _ := item["session_id"].(string)
	if sessionID == "" {
		t.Fatalf("expected session_id on item: %v", item)
	}

	itemID, _ := item["id"].(string)
	code, detail := evalAutoJSON(t, handler, http.MethodGet, "/v1/dashboard/eval/jobs/"+id+"/items/"+itemID, nil)
	if code != http.StatusOK {
		t.Fatalf("item detail status=%d body=%v", code, detail)
	}
	if detail["session"] == nil {
		t.Fatalf("missing session in detail: %v", detail)
	}
	loop, _ := detail["loop"].(map[string]any)
	if loop == nil {
		t.Fatalf("missing loop in detail: %v", detail)
	}
}

func TestEvalJobCancelEndpoint(t *testing.T) {
	handler := testEvalAutoHandler(t)
	code, body := evalAutoJSON(t, handler, http.MethodPost, "/v1/dashboard/eval/jobs", map[string]any{
		"title":    "取消冒烟",
		"case_ids": []string{"turn_plan_chat_definition"},
	})
	if code != http.StatusOK {
		t.Fatalf("create job status=%d body=%v", code, body)
	}
	job, _ := body["job"].(map[string]any)
	id, _ := job["id"].(string)
	if id == "" {
		t.Fatalf("missing job id: %v", body)
	}
	code, cancelBody := evalAutoJSON(t, handler, http.MethodPost, "/v1/dashboard/eval/jobs/"+id+"/cancel", nil)
	if code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%v", code, cancelBody)
	}
	got, _ := cancelBody["job"].(map[string]any)
	status, _ := got["status"].(string)
	if status == "" {
		t.Fatalf("cancel missing status: %v", cancelBody)
	}
}

func TestEvalAutoPageServesHTML(t *testing.T) {
	handler := testEvalAutoHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/v1/dashboard/eval/auto", nil)
	req.Header.Set("Authorization", "Bearer test-runtime-key")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type=%s", ct)
	}
	if !strings.Contains(rec.Body.String(), "自动测评") {
		t.Fatalf("html missing title")
	}
	if !strings.Contains(rec.Body.String(), "取消运行") {
		t.Fatalf("html missing cancel control")
	}
}
