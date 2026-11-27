package diagnoseskill_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func valued(id string, fields map[string]any) map[string]any {
	r := row(id, "Neutral", "neutral")
	for k, v := range fields {
		r[k] = v
	}
	return r
}

func rating(validity, kind, materiality string) map[string]any {
	return map[string]any{"validity": validity, "non_news_type": kind, "materiality": materiality, "value_reason": "这是价值判断依据。"}
}

func TestMacroNewsValueLabelsAndCodeOwnedEligibility(t *testing.T) {
	cases := []struct {
		name     string
		fields   map[string]any
		validity string
		high     bool
		eligible bool
	}{
		{"high", rating("news", "", "high"), "news", true, true},
		{"medium", rating("news", "", "medium"), "news", false, true},
		{"low", rating("NEWS", "", "Low"), "news", false, true},
		{"unrelated", rating("news", "", "none"), "news", false, false},
		{"quote", rating("non_news", "quote_page", ""), "non_news", false, false},
		{"quote with materiality", rating("non_news", "listing", "high"), "non_news", false, false},
	}
	for _, tc := range cases {
		raw, _ := json.Marshal(map[string]any{"items": []any{valued("a", tc.fields)}})
		good, bad := diagnoseskill.ParseMacroNewsReply(string(raw), []diagnoseskill.MacroNewsItem{{NewsID: "a", Title: "t"}})
		if len(bad) != 0 {
			t.Fatalf("%s: %v", tc.name, bad)
		}
		got := good["a"]
		if got.Validity != tc.validity || got.PoolEligible != tc.eligible || got.AnalysisStatus != "ok" ||
			got.ValueSource != "model" || got.ValueVersion != "v1" {
			t.Errorf("%s: %+v", tc.name, got)
		}
		if tc.validity == "non_news" && got.Materiality != "" {
			t.Errorf("%s: materiality must be empty for non_news", tc.name)
		}
		if tc.high {
			if got.JEVDirection != "neutral" || !strings.Contains(got.SummaryCN, "市场情绪：中性") {
				t.Errorf("%s: high-value items are analyzed: %+v", tc.name, got)
			}
			continue
		}
		// An item that will be discarded carries no analysis, even if the model sent one.
		if got.JEVDirection != "" || got.Attitude != "" || got.SummaryCN != "" || got.SummaryEN != "" ||
			got.JEVReason != "" || len(got.Tags) != 0 || len(got.RelatedSymbols) != 0 {
			t.Errorf("%s: no analysis for a discarded item: %+v", tc.name, got)
		}
	}
}

func TestMacroNewsShortReplyIsEnoughForDiscardedItems(t *testing.T) {
	short := reply(func(id string) map[string]any {
		r := rating("news", "", "low")
		r["news_id"] = id
		return r
	})
	fallbacks := 0
	fallback := func([]string) (string, error) { fallbacks++; return "{}", nil }
	decision := &fakeDecision{choice: "看多"}
	res, items := run(t, newsRequest(3), &scriptedLLM{replys: []func([]string) (string, error){short, fallback}}, decision.client(t))
	if fallbacks != 0 {
		t.Fatalf("fallbacks=%d", fallbacks)
	}
	for _, item := range items {
		if item.AnalysisStatus != "ok" || item.Materiality != "low" || item.JEVDirection != "" {
			t.Fatalf("item=%+v", item)
		}
	}
	// Nothing high value: no pool verdict, and certainly not a neutral one.
	if res["pool_status"] != "pending" || res["jev_direction"] != "" || len(decision.paths) != 0 {
		t.Fatalf("res=%v paths=%v", res, decision.paths)
	}
}

func TestMacroNewsMissingRatingAsksTheFallbackThenReportsUnrated(t *testing.T) {
	unrated := reply(func(id string) map[string]any { return valued(id, rating("huge", "x", "y")) })
	res, items := run(t, newsRequest(1), &scriptedLLM{replys: []func([]string) (string, error){unrated, bullish}}, nil)
	if items[0].AnalysisStatus != "ok" || items[0].Materiality != "high" || res["model"] != "fallback-model" {
		t.Fatalf("items=%+v", items)
	}
	_, items = run(t, newsRequest(1), &scriptedLLM{replys: []func([]string) (string, error){unrated}}, nil)
	if items[0].AnalysisStatus != "failed" || items[0].ErrorCode != "unrated" || items[0].Validity != "" {
		t.Fatalf("items=%+v", items)
	}
	down := func([]string) (string, error) { return "", errors.New("503") }
	_, items = run(t, newsRequest(1), &scriptedLLM{replys: []func([]string) (string, error){down}}, nil)
	if items[0].AnalysisStatus != "failed" || items[0].ErrorCode != "" {
		t.Fatalf("a failed call is not an unrated item: %+v", items)
	}
}

func TestMacroNewsOnlyHighValueItemsReachThePool(t *testing.T) {
	build := reply(func(id string) map[string]any {
		if id == "n0" {
			return row(id, "Bearish", "short")
		}
		return valued(id, rating("news", "", "medium"))
	})
	decision := &fakeDecision{choice: "看空"}
	res, _ := run(t, newsRequest(3), &scriptedLLM{replys: []func([]string) (string, error){build}}, decision.client(t))
	if res["pool_status"] != "ok" || strings.Contains(decision.choiceBody, "新闻 2") {
		t.Fatalf("res=%v body=%s", res, decision.choiceBody)
	}
}

func TestMacroNewsRelatedSymbolsMustBeMentioned(t *testing.T) {
	symbol := func(code, mention string) map[string]any {
		return map[string]any{"symbol": code, "name": "名称", "market": "hk", "mention": mention}
	}
	r := row("a", "Bullish", "long")
	r["related_symbols"] = []any{
		symbol("00700.hk", "腾讯"), symbol("00700.HK", "腾讯"), symbol("AAPL", "apple"),
		symbol("800000.HK", "恒生指数"), // the pool's index, not in the story
		symbol("09988.HK", ""), "junk", symbol("", "腾讯"),
	}
	raw, _ := json.Marshal(map[string]any{"items": []any{r}})
	good, bad := diagnoseskill.ParseMacroNewsReply(string(raw), []diagnoseskill.MacroNewsItem{
		{NewsID: "a", Title: "腾讯回购", Snippet: "Apple 同日公告"},
	})
	got := good["a"].RelatedSymbols
	if len(bad) != 0 || len(got) != 2 || got[0] != (diagnoseskill.MacroNewsSymbol{Symbol: "00700.HK", Name: "名称", Market: "HK"}) || got[1].Symbol != "AAPL" {
		t.Fatalf("symbols=%+v bad=%v", got, bad)
	}
	// A malformed list never fails the item.
	r["related_symbols"] = "AAPL"
	raw, _ = json.Marshal(map[string]any{"items": []any{r}})
	good, bad = diagnoseskill.ParseMacroNewsReply(string(raw), []diagnoseskill.MacroNewsItem{{NewsID: "a", Title: "t"}})
	if len(bad) != 0 || good["a"].RelatedSymbols == nil || len(good["a"].RelatedSymbols) != 0 {
		t.Fatalf("good=%+v bad=%v", good, bad)
	}
}

func TestMacroNewsRollupOnly(t *testing.T) {
	req := newsRequest(0)
	req.RollupOnly = true
	req.PriorVerdicts = []diagnoseskill.MacroNewsPriorVerdict{
		{NewsID: "p1", Title: "利空一", JEVDirection: "short", JEVReason: "理由"},
	}
	decision := &fakeDecision{choice: "看空"}
	llm := &scriptedLLM{summaries: []string{poolSummary}}
	res, items := run(t, req, llm, decision.client(t))
	if len(items) != 0 || llm.calls != 0 || llm.summaryCalls != 1 {
		t.Fatalf("items=%d calls=%d summaries=%d", len(items), llm.calls, llm.summaryCalls)
	}
	if res["pool_status"] != "ok" || res["jev_direction"] != "short" || res["jev_reason"] != poolSummary {
		t.Fatalf("res=%v", res)
	}
	// Without a synthesis model the direction still stands.
	res, _ = run(t, req, nil, decision.client(t))
	if res["jev_direction"] != "short" || res["reason_status"] != "failed" {
		t.Fatalf("res=%v", res)
	}
	// A failed Choice is pending with no direction.
	res, _ = run(t, req, llm, (&fakeDecision{fail: true}).client(t))
	if res["pool_status"] != "pending" || res["jev_direction"] != "" || res["pool_error"] == "" {
		t.Fatalf("res=%v", res)
	}
	for name, mutate := range map[string]func(*diagnoseskill.MacroNewsRequest){
		"with items": func(r *diagnoseskill.MacroNewsRequest) { r.Items = newsRequest(1).Items },
		"no priors":  func(r *diagnoseskill.MacroNewsRequest) { r.PriorVerdicts = nil },
		"skip too":   func(r *diagnoseskill.MacroNewsRequest) { r.SkipPoolJEV = true },
	} {
		bad := req
		mutate(&bad)
		if diagnoseskill.ValidateMacroNewsRequest(bad) == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

type promptRecorder struct{ system string }

func (p *promptRecorder) Complete(_ context.Context, system, _ string, _ func(string) error) (string, error) {
	p.system = system
	return "", context.Canceled
}

func TestMacroNewsPromptOrderAndRules(t *testing.T) {
	rec := &promptRecorder{}
	_, _ = diagnoseskill.RunMacroNews(context.Background(), diagnoseskill.MacroNewsRunInput{Request: newsRequest(1), LLM: rec})
	at := func(s string) int {
		i := strings.Index(rec.system, s)
		if i < 0 {
			t.Fatalf("prompt lacks %q", s)
		}
		return i
	}
	if !(at("第一步 validity") < at("第二步 materiality") && at("第二步 materiality") < at("第三步")) {
		t.Fatal("decision order must be: news?, materiality, direction")
	}
	if !(at("- validity\n") < at("- value_reason\n") && at("- value_reason\n") < at("- title_cn") && at("- title_cn") < at("- attitude")) {
		t.Fatal("value fields come first in the output")
	}
	for _, rule := range []string{
		"whether", "网站名称本身不能决定", "方向不决定重要性", "news 加 low", "其他条目只输出",
		"related_symbols", "不要按监控池补充", strategydev.MacroNewsJEVPrinciples,
		"1. 行情页", "2. 市场综述", "3. 例行公告", "4. 政策意外", "5. 关键字误命中",
	} {
		at(rule)
	}
	if strings.Contains(rec.system, "pool_eligible") || strings.Contains(rec.system, "照常输出下面的全部字段") {
		t.Fatal("no eligibility, and no full analysis of discarded items")
	}
}
