package diagnoseskill_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/diagnoseskill"
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
		eligible bool
	}{
		{"high", rating("news", "", "high"), "news", true},
		{"low", rating("NEWS", "", "Low"), "news", true},
		{"unrelated", rating("news", "", "none"), "news", false},
		{"quote", rating("non_news", "quote_page", ""), "non_news", false},
		{"quote with materiality", rating("non_news", "listing", "high"), "non_news", false},
		{"missing", map[string]any{}, "", true},
		{"bad validity", rating("spam", "", "low"), "", true},
		{"bad materiality", rating("news", "", "critical"), "", true},
		{"missing materiality", rating("news", "", ""), "", true},
		{"bad type", rating("non_news", "junk", ""), "", true},
		{"numeric materiality", map[string]any{"validity": "news", "materiality": 90, "value_reason": "理由。"}, "", true},
		{"wrong types everywhere", map[string]any{"validity": true, "non_news_type": []string{"x"}, "materiality": map[string]any{}, "value_reason": 3}, "", true},
		{"english reason", map[string]any{"validity": "news", "materiality": "low", "value_reason": "routine"}, "", true},
		{"no reason", map[string]any{"validity": "news", "materiality": "low"}, "", true},
		{"model cannot set eligibility", map[string]any{"pool_eligible": false}, "", true},
	}
	for _, tc := range cases {
		raw, _ := json.Marshal(map[string]any{"items": []any{valued("a", tc.fields)}})
		good, bad := diagnoseskill.ParseMacroNewsReply(string(raw), []diagnoseskill.MacroNewsItem{{NewsID: "a", Title: "t"}})
		if len(bad) != 0 {
			t.Fatalf("%s: a value field must never fail an item: %v", tc.name, bad)
		}
		got := good["a"]
		if got.Validity != tc.validity || got.PoolEligible != tc.eligible || got.AnalysisStatus != "ok" {
			t.Errorf("%s: %+v", tc.name, got.MacroNewsValue)
		}
		// Record-only: the direction and sentiment are kept whatever the rating.
		if got.JEVDirection != "neutral" || got.Attitude != "Neutral" || !strings.Contains(got.SummaryCN, "市场情绪：中性") {
			t.Errorf("%s: analysis must be intact: %+v", tc.name, got)
		}
		if tc.validity == "" {
			if got.MacroNewsValue != (diagnoseskill.MacroNewsValue{PoolEligible: true}) {
				t.Errorf("%s: unrated must have empty labels: %+v", tc.name, got.MacroNewsValue)
			}
		} else if got.ValueSource != "model" || got.ValueVersion != "v1" {
			t.Errorf("%s: %+v", tc.name, got.MacroNewsValue)
		}
		if tc.validity == "non_news" && got.Materiality != "" {
			t.Errorf("%s: materiality must be empty for non_news", tc.name)
		}
	}
}

func TestMacroNewsNonNewsWithoutDirectionStaysEmpty(t *testing.T) {
	junk := rating("non_news", "quote_page", "")
	junk["attitude"], junk["jev_direction"], junk["jev_reason"] = "", "", ""
	real := rating("news", "", "low")
	real["attitude"], real["jev_direction"] = "", ""
	broken := rating("non_news", "quote_page", "")
	broken["jev_direction"] = "sideways"
	raw, _ := json.Marshal(map[string]any{"items": []any{valued("junk", junk), valued("real", real), valued("broken", broken)}})
	good, bad := diagnoseskill.ParseMacroNewsReply(string(raw), []diagnoseskill.MacroNewsItem{
		{NewsID: "junk", Title: "t"}, {NewsID: "real", Title: "t"}, {NewsID: "broken", Title: "t"},
	})
	got, ok := good["junk"]
	if !ok || got.Attitude != "" || got.JEVDirection != "" || got.JEVReason != "" {
		t.Fatalf("junk=%+v bad=%v", got, bad)
	}
	if strings.Contains(got.SummaryCN, "市场情绪") || strings.Contains(got.SummaryEN, "Sentiment") {
		t.Fatalf("no sentiment line without a direction: %q %q", got.SummaryCN, got.SummaryEN)
	}
	if bad["real"] == "" || bad["broken"] == "" {
		t.Fatalf("news without a direction, and an invalid direction, still fail: %v", bad)
	}
}

func TestMacroNewsValueNeverTriggersFallback(t *testing.T) {
	calls := 0
	primary := reply(func(id string) map[string]any { return valued(id, rating("huge", "x", "y")) })
	fallback := func([]string) (string, error) { calls++; return "{}", nil }
	decision := &fakeDecision{choice: "看多"}
	res, items := run(t, newsRequest(3), &scriptedLLM{replys: []func([]string) (string, error){primary, fallback}}, decision.client(t))
	if calls != 0 || res["model"] != "primary-model" {
		t.Fatalf("fallback calls=%d model=%v", calls, res["model"])
	}
	for _, item := range items {
		if item.AnalysisStatus != "ok" || item.Validity != "" || !item.PoolEligible {
			t.Fatalf("item=%+v", item)
		}
	}
}

func TestMacroNewsRecordOnlyKeepsIneligibleItemsInThePool(t *testing.T) {
	decision := &fakeDecision{choice: "看多"}
	llm := &scriptedLLM{replys: []func([]string) (string, error){reply(func(id string) map[string]any {
		return valued(id, rating("non_news", "listing", ""))
	})}}
	res, items := run(t, newsRequest(3), llm, decision.client(t))
	if res["pool_status"] != "ok" || items[0].PoolEligible {
		t.Fatalf("res=%v item=%+v", res["pool_status"], items[0])
	}
	if !strings.Contains(decision.choiceBody, "新闻 3") {
		t.Fatal("all three must reach Decision Choice while ratings are record-only")
	}
}

func TestMacroNewsDirectionlessItemsSkipTheRollupNotCountAsNeutral(t *testing.T) {
	build := reply(func(id string) map[string]any {
		if id == "n0" {
			return valued(id, rating("news", "", "high"))
		}
		junk := rating("non_news", "quote_page", "")
		junk["attitude"], junk["jev_direction"] = "", ""
		return valued(id, junk)
	})
	decision := &fakeDecision{choice: "看空"}
	res, _ := run(t, newsRequest(3), &scriptedLLM{replys: []func([]string) (string, error){build}}, decision.client(t))
	// One directional item of one: the two without a direction leave the threshold too.
	if res["pool_status"] != "ok" || strings.Contains(decision.choiceBody, "新闻 2") {
		t.Fatalf("res=%v body=%s", res, decision.choiceBody)
	}

	onlyJunk := &fakeDecision{choice: "看多"}
	req := newsRequest(2)
	req.Items[0].NewsID, req.Items[1].NewsID = "x1", "x2"
	res, _ = run(t, req, &scriptedLLM{replys: []func([]string) (string, error){build}}, onlyJunk.client(t))
	if res["pool_status"] != "pending" || res["jev_direction"] != "" || len(onlyJunk.paths) != 0 {
		t.Fatalf("res=%v paths=%v", res, onlyJunk.paths)
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
		"whether", "网站名称本身不能决定", "方向不决定重要性", "news 加 low", "每一条都照常输出下面的全部字段",
		"1. 行情页", "2. 市场综述", "3. 例行公告", "4. 政策意外", "5. 关键字误命中",
	} {
		at(rule)
	}
	if strings.Contains(rec.system, "pool_eligible") {
		t.Fatal("the model must not be asked for eligibility")
	}
}
