package runtimeapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/eval"
)

func TestEvalAutoClarifyNoScriptFirstBypass(t *testing.T) {
	h := NewHandler(&app.App{Config: &config.AppConfig{}}, "")
	body, _ := json.Marshal(map[string]any{
		"question":         "请选择策略",
		"choices":          []string{"SAR+MACD", "RSI 触发型"},
		"clarify_defaults": []string{"SAR+MACD"},
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/dashboard/eval/auto-clarify", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	h.evalAutoClarify(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp["source"] == "script" {
		t.Fatalf("script-first bypass should be removed, source=%v", resp["source"])
	}
	if int(resp["auto_pick_seconds"].(float64)) != eval.DefaultClarifyAutoPickSeconds {
		t.Fatalf("auto_pick_seconds=%v", resp["auto_pick_seconds"])
	}
}
