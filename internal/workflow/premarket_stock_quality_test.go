package workflow_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ghsemail/GeeGooAgent/internal/memory"
	"github.com/ghsemail/GeeGooAgent/internal/workflow"
)

func TestBuildCreateReportArgsIncludesSupportResistance(t *testing.T) {
	w := &memory.PreMarketWorking{
		MarketReportResult: "neutral",
		BotCodes:           []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {
				Code:      "00700.HK",
				StockName: "腾讯控股",
				Attitude:  "bearish",
				WeeklyAnalysisRef: `关键点位：
- 上方压力：455~460
- 下方支撑：436~440`,
				CapitalDistributionSummary: "**简要解读**：小单净流入而主力偏弱，散户接盘特征需警惕。",
			},
		},
	}
	args := workflow.BuildCreateReportArgsContext(context.Background(), w, "00700.HK")
	if args["support"] == nil || args["resistance"] == nil {
		t.Fatalf("support/resistance missing: %+v", args)
	}
	if args["suggestion"] == "sell" {
		t.Fatalf("divergent capital + neutral market should not suggest sell, got %v", args["suggestion"])
	}
	reason := args["reason"].(string)
	if strings.Contains(reason, "证据已纳入") {
		t.Fatalf("boilerplate reason leaked: %s", reason)
	}
	if len([]rune(reason)) < 80 {
		t.Fatalf("reason too short: %s", reason)
	}
	summary := args["summary"].(string)
	if strings.Contains(summary, "新闻综述") {
		t.Fatalf("summary should be one-liner, got: %s", summary)
	}
}

func TestBuildCreateReportArgsSubstantiveFallbackOnSynthError(t *testing.T) {
	ctx := workflow.ContextWithSynthesizer(context.Background(), &failingSynthesizer{})
	w := &memory.PreMarketWorking{
		MarketReportResult: "neutral",
		BotCodes:           []memory.BotStock{{Code: "00700.HK", StockName: "腾讯控股"}},
		Stocks: map[string]memory.StockWorkspace{
			"00700.HK": {
				Code: "00700.HK", StockName: "腾讯控股", Attitude: "bearish",
				WeeklyAnalysisRef:          "结论：震荡偏弱，支撑 436，阻力 460。",
				CapitalDistributionSummary: "**简要解读**：小单净流入而主力偏弱。",
			},
		},
	}
	args := workflow.BuildCreateReportArgsContext(ctx, w, "00700.HK")
	if args["suggestion"] != "hold" {
		t.Fatalf("suggestion=%v want hold", args["suggestion"])
	}
}

func TestOneLineTruncatesByRune(t *testing.T) {
	out := memory.OneLine("市场背景 恒指近 5 日累计下跌约 1.38% 后于 25,453 附近企稳", 20)
	if len([]rune(out)) > 20 {
		t.Fatalf("expected <=20 runes, got %d: %q", len([]rune(out)), out)
	}
	if !strings.HasSuffix(out, "...") {
		t.Fatalf("expected ellipsis suffix: %q", out)
	}
}
