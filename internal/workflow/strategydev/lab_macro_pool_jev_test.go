package strategydev_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func mockPoolJEVServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/decision/choice":
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev-pool","answers":{"direction":{"type":"choice","choice":"看多","probabilities":{"看多":0.7,"看空":0.2,"不明确":0.1},"confidence":0.72}}}}`))
		case "/v1/decision/chat":
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev-pool","content":"## 整体结论\n看多，置信 0.72。\n1. BTC 结构偏多\n2. ETH 动能改善"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestResolveMacroPoolJEVChoiceAndReason(t *testing.T) {
	srv := mockPoolJEVServer(t)
	defer srv.Close()
	jev := strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k", Model: "jev-pool"})
	symbols := []strategydev.MacroPoolSymbolInput{
		{Symbol: "BTC-USD.CRYPTO", ResultID: "rid-btc", JEVDirection: "long", JEVReason: "BTC 偏多依据足够长用于测试", Confidence: 0.8},
		{Symbol: "ETH-USD.CRYPTO", ResultID: "rid-eth", JEVDirection: "long", JEVReason: "ETH 同步改善依据足够长", Confidence: 0.75},
	}
	out := strategydev.ResolveMacroPoolJEV(context.Background(), jev, "crypto", "加密", symbols)
	if !out.OK {
		t.Fatalf("ok=false err=%s", out.Error)
	}
	if out.Direction != strategydev.LabJEVUnifiedBull {
		t.Fatalf("direction=%s", out.Direction)
	}
	if out.Confidence < 0.7 {
		t.Fatalf("confidence=%v", out.Confidence)
	}
	if !strings.Contains(out.RecommendedReason, "整体结论") {
		t.Fatalf("reason=%q", out.RecommendedReason)
	}
}

func TestBuildMacroPoolJEVStateIncludesReasonAndConfidence(t *testing.T) {
	symbols := []strategydev.MacroPoolSymbolInput{
		{Symbol: "BTC-USD.CRYPTO", JEVDirection: "long", JEVReason: "链上资金流改善", Confidence: 0.81},
	}
	state := strategydev.BuildMacroPoolJEVState("加密", symbols)
	if !strings.Contains(state, "依据摘要") || !strings.Contains(state, "链上资金流") {
		t.Fatalf("state missing reason facts: %s", state)
	}
	if !strings.Contains(state, "0.810") {
		t.Fatalf("state missing confidence: %s", state)
	}
}

func TestMacroPoolSymbolConflictCapsConfidence(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/decision/choice" {
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev","answers":{"direction":{"choice":"看多","probabilities":{"看多":0.9,"看空":0.05,"不明确":0.05},"confidence":0.9}}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\n冲突下看多。"}}`))
	}))
	defer srv.Close()
	jev := strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k"})
	symbols := []strategydev.MacroPoolSymbolInput{
		{Symbol: "A", ResultID: "r1", JEVDirection: "long", JEVReason: "A long reason text here ok"},
		{Symbol: "B", ResultID: "r2", JEVDirection: "short", JEVReason: "B short reason text here ok"},
	}
	out := strategydev.ResolveMacroPoolJEV(context.Background(), jev, "crypto", "", symbols)
	if !out.OK {
		t.Fatal(out.Error)
	}
	if !out.Conflict {
		t.Fatal("expected conflict")
	}
	if out.Confidence > 0.75+1e-9 {
		t.Fatalf("confidence=%v", out.Confidence)
	}
}

func TestMacroPoolConflictPersistsWhenChoiceUnclear(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/decision/choice" {
			_, _ = w.Write([]byte(`{"code":0,"data":{"model":"jev","answers":{"direction":{"choice":"不明确","probabilities":{"看多":0.35,"看空":0.35,"不明确":0.3},"confidence":0.88}}}}`))
			return
		}
		_, _ = w.Write([]byte(`{"code":0,"data":{"content":"## 整体结论\n不明确。"}}`))
	}))
	defer srv.Close()
	jev := strategydev.NewJevClient(strategydev.JevConfig{BaseURL: srv.URL, APIKey: "k"})
	symbols := []strategydev.MacroPoolSymbolInput{
		{Symbol: "A", ResultID: "r1", JEVDirection: "long", JEVReason: "A long reason text here ok"},
		{Symbol: "B", ResultID: "r2", JEVDirection: "short", JEVReason: "B short reason text here ok"},
	}
	out := strategydev.ResolveMacroPoolJEV(context.Background(), jev, "crypto", "", symbols)
	if !out.OK {
		t.Fatal(out.Error)
	}
	if out.Direction != strategydev.LabJEVUnifiedUnclear {
		t.Fatalf("direction=%s", out.Direction)
	}
	if !out.Conflict {
		t.Fatal("conflict must stay true when inputs disagree even if choice is 不明确")
	}
	if out.Confidence > 0.75+1e-9 {
		t.Fatalf("confidence=%v", out.Confidence)
	}
}