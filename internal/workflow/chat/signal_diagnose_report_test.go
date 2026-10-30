package chat

import (
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/slots"
)

func TestRenderSignalDiagnoseReportCompactNotKBDraft(t *testing.T) {
	flow := flowWithFullEval()
	flow.EvalJudgment = "结论：样本内买点有效，注意卖段稀疏。"
	flow.KBDraft = assembleSignalDiagnoseDoc(flow, flow.EvalJudgment)
	flow.KnowledgeID = "kb-1"
	flow.KnowledgeTitle = "SAR · 腾讯控股 · 信号诊断"
	report := renderSignalDiagnoseReport(flow)
	if strings.Contains(report, flow.KBDraft) {
		t.Fatal("chat report must not echo full KB draft")
	}
	for _, forbidden := range []string{
		"命中率与有效性怎么算",
		"汇总 · Strict",
		"持有K线",
		"买入规则",
		"样例触发",
	} {
		if strings.Contains(report, forbidden) {
			t.Fatalf("chat report must omit %q:\n%s", forbidden, report)
		}
	}
	if !strings.Contains(report, "Agent 评价") || !strings.Contains(report, "结论：样本内买点有效") {
		t.Fatalf("missing judgment:\n%s", report)
	}
	if !strings.Contains(report, "kb-1") {
		t.Fatalf("missing knowledge id:\n%s", report)
	}
	if len(report) >= len(flow.KBDraft) {
		t.Fatalf("expected compact report (%d) << kb draft (%d)", len(report), len(flow.KBDraft))
	}
}

func TestAssembleSignalDiagnoseDocKeepsFullDetail(t *testing.T) {
	flow := flowWithFullEval()
	doc := assembleSignalDiagnoseDoc(flow, "Agent 综合判断正文。")
	for _, want := range []string{
		"命中率与有效性怎么算",
		"汇总 · Strict",
		"Agent 评价",
		"Agent 综合判断正文。",
	} {
		if !strings.Contains(doc, want) {
			t.Fatalf("KB doc missing %q:\n%s", want, doc)
		}
	}
}

func flowWithFullEval() *Flow {
	return &Flow{
		StrategyQuery: "SAR",
		CatalogType:   "index",
		StockName:     "腾讯控股",
		StockCode:     "00700.HK",
		MonthsBack:    3,
		ProbeBuyHits:  15,
		ProbeSellHits: 16,
		Strategies:    []StrategyItem{{Frequency: "60m"}},
		SignalEval: slots.SignalEpisodeEval{
			Method: "close",
			BuyEpisodes: slots.SignalEpisodeMetrics{
				CompleteCount: 2, HitCount: 1, HitRate: 0.5,
			},
		},
		DiagnoseRaw: map[string]any{
			"buy_rules": []any{map[string]any{
				"index": "SAR", "type": "signal", "trigger_count": 15,
				"sample_reasons": []any{map[string]any{"time": "2026-01-01", "reason": "flip"}},
			}},
		},
	}
}
