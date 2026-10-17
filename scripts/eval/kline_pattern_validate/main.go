// Command kline_pattern_validate runs probe_bot_signal_series on live symbols and prints
// K-line pattern display JSON snippets for manual review.
//
// Env: SIGNAL_API_URL (default http://146.56.225.252:3200), SIGNAL_API_KEY (required)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
	"github.com/ghsemail/GeeGooAgent/internal/workflow/strategydev"
)

func main() {
	key := strings.TrimSpace(os.Getenv("SIGNAL_API_KEY"))
	if key == "" {
		fmt.Fprintln(os.Stderr, "SIGNAL_API_KEY required")
		os.Exit(1)
	}
	url := envOr("SIGNAL_API_URL", "http://146.56.225.252:3200")
	client := mcp.NewClient(url, key, mcp.Options{
		AllowedHosts: []string{"146.56.225.252", "127.0.0.1", "localhost"},
	})
	reg := tools.NewRegistry()
	tools.RegisterAll(reg, tools.Deps{HTTP: tools.HTTPBackends{SignalAPI: client}})

	type caseSpec struct {
		Code  string
		Freq  string
		Label string
	}
	cases := []caseSpec{
		{Code: "00700.HK", Freq: "daily", Label: "腾讯日线"},
		{Code: "AAPL.US", Freq: "60m", Label: "苹果60m"},
	}
	if extra := strings.TrimSpace(os.Getenv("KLINE_VALIDATE_CODE")); extra != "" {
		cases = append(cases, caseSpec{Code: extra, Freq: envOr("KLINE_VALIDATE_FREQ", "daily"), Label: "extra"})
	}

	ctx := context.Background()
	run := func(c context.Context, name string, args map[string]any) tools.Result {
		return reg.Execute(tools.CallRequest{Name: name, Arguments: args}, tools.Context{})
	}

	for _, cs := range cases {
		fmt.Printf("\n======== %s %s ========\n", cs.Label, cs.Code)
		probeArgs := map[string]any{
			"code": cs.Code, "frequency": strategydev.NormalizeFrequency(cs.Freq),
			"months_back": 3,
			"buy_signal":  []any{map[string]any{"index": "RSIThrehold", "type": "signal", "param": map[string]any{"period": 14, "threholdBuy": 30, "threholdSell": 70}}},
			"sell_signal": []any{map[string]any{"index": "RSIThrehold", "type": "signal", "param": map[string]any{"period": 14, "threholdBuy": 30, "threholdSell": 70}}},
		}
		res := run(ctx, "probe_bot_signal_series", probeArgs)
		if res.Status != tools.StatusOK {
			fmt.Printf("probe failed: %s\n", res.Summary)
			continue
		}
		bars, _ := res.Data["bars"].([]any)
		buyM, _ := res.Data["buy_merged"].([]any)
		sellM, _ := res.Data["sell_merged"].([]any)
		fmt.Printf("bars=%d pattern_hits=%v buy_hits=%v\n", len(bars), lenAny(res.Data["pattern_hits"]), res.Data["buy_hits"])

		mtf := strategydev.BuildKlinePatternMTFForDiagnose(ctx, run, cs.Code, cs.Freq, 3, bars)
		patIn := strategydev.NewKlinePatternDisplayInput(cs.Freq, bars, res.Data, mtf)
		buyTr, sellTr := strategydev.CollectEpisodeTriggersFromMerged(bars, buyM, sellM)
		strategydev.AttachKlineDisplayToProbeTriggers(buyTr, sellTr, patIn)

		show := func(side string, trs []strategydev.EpisodeTriggerRecord) {
			if len(trs) == 0 {
				fmt.Printf("no %s triggers\n", side)
				return
			}
			tr := trs[0]
			fmt.Printf("%s trigger[0] idx=%d time=%s summary=%q bias=%+v\n", side, tr.Index, tr.Time, tr.KlinePatternSummary, tr.KlinePatternBias)
			if tr.KlinePatternThreeBar != nil {
				for _, b := range tr.KlinePatternThreeBar.Bars {
					fmt.Printf("  %s idx=%d omitted=%v close=%.2f patterns=%q\n", b.Role, b.Index, b.Omitted, b.Close, b.KlinePatternSummary)
				}
			}
			if tr.KlinePatternMTF != nil {
				fmt.Printf("  MTF note=%s primary=%+v tfs=%d\n", tr.KlinePatternMTF.Note, tr.KlinePatternMTF.PrimaryBias, len(tr.KlinePatternMTF.Timeframes))
				for _, tf := range tr.KlinePatternMTF.Timeframes {
					fmt.Printf("    %s aligned=%d bias=%+v three_bar=%v\n", tf.Label, tf.AlignedIndex, tf.Bias, tf.ThreeBar != nil)
				}
			}
			if len(bars) > tr.Index+3 {
				row, _ := bars[tr.Index+3].(map[string]any)
				fmt.Printf("  forward+3 close=%.2f (walk-forward sanity, not prediction)\n", floatAny(row["close"]))
			}
			raw, _ := json.MarshalIndent(map[string]any{
				"kline_patterns": tr.KlinePatterns, "kline_pattern_bias": tr.KlinePatternBias,
				"kline_pattern_three_bar": tr.KlinePatternThreeBar, "kline_pattern_mtf": tr.KlinePatternMTF,
			}, "  ", "  ")
			fmt.Printf("  json snippet:\n  %s\n", string(raw))
		}
		show("buy", buyTr)
		if len(sellTr) > 0 {
			show("sell", sellTr)
		}
	}
}

func envOr(k, fb string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return fb
}

func lenAny(v any) int {
	if s, ok := v.([]any); ok {
		return len(s)
	}
	return 0
}

func floatAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	default:
		return 0
	}
}
