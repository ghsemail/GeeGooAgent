package diagnoseskill_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func overviewRequest() diagnoseskill.MacroNewsOverviewRequest {
	pool := func(id, direction string) diagnoseskill.MacroNewsOverviewPool {
		return diagnoseskill.MacroNewsOverviewPool{
			PoolID: id, JEVDirection: direction, JEVReason: "## 整体结论\n" + id + " 结论", Bullish: 3, HighValueCount: 3,
			Items: []diagnoseskill.MacroNewsOverviewItem{{Title: id + " 要闻", JEVDirection: direction, JEVReason: "要闻理由"}},
		}
	}
	return diagnoseskill.MacroNewsOverviewRequest{
		Date: "2026-11-27", NodeID: "us-hk", RunID: "r1",
		Pools:        []diagnoseskill.MacroNewsOverviewPool{pool("hk", "long"), pool("us", "long")},
		MissingPools: []string{"cn_a", "crypto"},
	}
}

const overviewSummary = "## 整体结论\n\n方向看空，置信度中等。核心依据：跨市场流动性收紧。结论未覆盖 A股池与虚拟货币池。"

func TestMacroNewsOverview(t *testing.T) {
	decision := &fakeDecision{choice: "看空"}
	llm := &scriptedLLM{summaries: []string{overviewSummary}}
	res, err := diagnoseskill.RunMacroNewsOverview(context.Background(), diagnoseskill.MacroNewsOverviewRunInput{
		Request: overviewRequest(), LLM: llm, JEV: decision.client(t),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Every pool is bullish; Choice still decides and no pools are counted.
	if res["jev_direction"] != "short" || res["jev_reason"] != overviewSummary || res["reason_status"] != "ok" {
		t.Fatalf("res=%v", res)
	}
	if res["date"] != "2026-11-27" || res["node_id"] != "us-hk" || len(res["choices"].([]map[string]any)) != 3 {
		t.Fatalf("res=%v", res)
	}
	if got := res["missing_pools"].([]string); strings.Join(got, ",") != "cn_a,crypto" {
		t.Fatalf("missing=%v", got)
	}
	if strings.Join(decision.paths, ",") != "/v1/decision/choice" {
		t.Fatalf("paths=%v", decision.paths)
	}
	for _, want := range []string{"A股池、虚拟货币池", "港股池", "hk 要闻", "us 结论", "risk-on"} {
		if !strings.Contains(decision.choiceBody, want) {
			t.Errorf("Choice state lacks %q", want)
		}
	}
	for _, rule := range []string{"## 整体结论", "不得改变、重推或质疑该方向", "missing_pools", strategydev.MacroNewsJEVPrinciples} {
		if !strings.Contains(llm.summarySystem, rule) {
			t.Errorf("summary prompt lacks %q", rule)
		}
	}
	var payload struct {
		RecommendedChoice string `json:"recommended_choice"`
		Pools             []struct {
			Pool      string           `json:"pool"`
			Headlines []map[string]any `json:"headlines"`
		} `json:"pools"`
		MissingPools []string `json:"missing_pools"`
	}
	if err := json.Unmarshal([]byte(llm.summaryUser), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.RecommendedChoice != "看空" || len(payload.Pools) != 2 || len(payload.MissingPools) != 2 {
		t.Fatalf("payload=%+v", payload)
	}
}

func TestMacroNewsOverviewFailures(t *testing.T) {
	// A failed summary keeps the direction and says so.
	res, err := diagnoseskill.RunMacroNewsOverview(context.Background(), diagnoseskill.MacroNewsOverviewRunInput{
		Request: overviewRequest(), LLM: &scriptedLLM{}, JEV: (&fakeDecision{choice: "看多"}).client(t),
	})
	if err != nil || res["jev_direction"] != "long" || res["reason_status"] != "failed" ||
		!strings.HasPrefix(res["jev_reason"].(string), "理由总结失败：") {
		t.Fatalf("res=%v err=%v", res, err)
	}
	// A failed Choice is an error, not a neutral overview.
	if _, err := diagnoseskill.RunMacroNewsOverview(context.Background(), diagnoseskill.MacroNewsOverviewRunInput{
		Request: overviewRequest(), LLM: &scriptedLLM{}, JEV: (&fakeDecision{fail: true}).client(t),
	}); err == nil {
		t.Fatal("expected an error")
	}
	for name, mutate := range map[string]func(*diagnoseskill.MacroNewsOverviewRequest){
		"bad date":      func(r *diagnoseskill.MacroNewsOverviewRequest) { r.Date = "x" },
		"no pools":      func(r *diagnoseskill.MacroNewsOverviewRequest) { r.Pools = nil },
		"unknown pool":  func(r *diagnoseskill.MacroNewsOverviewRequest) { r.Pools[0].PoolID = "fx" },
		"duplicate":     func(r *diagnoseskill.MacroNewsOverviewRequest) { r.Pools[1].PoolID = "hk" },
		"no direction":  func(r *diagnoseskill.MacroNewsOverviewRequest) { r.Pools[0].JEVDirection = "" },
		"missing twice": func(r *diagnoseskill.MacroNewsOverviewRequest) { r.MissingPools = []string{"hk"} },
		"too many items": func(r *diagnoseskill.MacroNewsOverviewRequest) {
			r.Pools[0].Items = make([]diagnoseskill.MacroNewsOverviewItem, 13)
		},
	} {
		req := overviewRequest()
		mutate(&req)
		if diagnoseskill.ValidateMacroNewsOverviewRequest(req) == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}
