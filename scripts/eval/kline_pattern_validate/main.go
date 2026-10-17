// Command kline_pattern_validate runs probe_bot_signal_series on live symbols and prints
// K-line pattern display JSON snippets for manual review.
//
// Fixed symbols: 00700.HK (腾讯 60m), SpaceX SPCX.US (5m, fallback 60m).
//
// Env: SIGNAL_API_URL, SIGNAL_API_KEY, MCP_TOKEN (for search_code / full probe payload)
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
	mcpToken := strings.TrimSpace(os.Getenv("MCP_TOKEN"))
	if mcpToken == "" {
		fmt.Fprintln(os.Stderr, "MCP_TOKEN required (search_code + non-compact probe payload)")
		os.Exit(1)
	}
	url := envOr("SIGNAL_API_URL", "http://146.56.225.252:3200")
	client := mcp.NewClient(url, key, mcp.Options{
		AllowedHosts: []string{"146.56.225.252", "127.0.0.1", "localhost"},
	})
	reg := tools.NewRegistry()
	tools.RegisterAll(reg, tools.Deps{HTTP: tools.HTTPBackends{SignalAPI: client, SignalCatalog: client}})

	toolCtx := tools.Context{MCPToken: mcpToken, FullCatalogPayload: true}
	ctx := context.Background()
	var run strategydev.ToolRunner = func(c context.Context, name string, args map[string]any) tools.Result {
		return reg.Execute(tools.CallRequest{Name: name, Arguments: args}, toolCtx)
	}

	spacexCode := resolveSpaceXCode(ctx, run)
	fmt.Printf("SpaceX symbol: %s (probe 固定 SPCX.US)\n", spacexCode)

	cases := []struct {
		Code  string
		Freq  string
		Label string
		Note  string
	}{
		{Code: "00700.HK", Freq: "60m", Label: "腾讯控股"},
		{Code: spacexCode, Freq: "5m", Label: "SpaceX"},
	}

	for _, cs := range cases {
		ok, bars := runCase(ctx, run, cs.Label, cs.Code, cs.Freq, "")
		if cs.Label == "SpaceX" && (!ok || bars == 0) {
			fmt.Println("SpaceX 5m 无有效 bars，回退 60m …")
			runCase(ctx, run, cs.Label, cs.Code, "60m", "5m 无数据，已回退 60m")
		}
	}
}

func resolveSpaceXCode(ctx context.Context, run strategydev.ToolRunner) string {
	const code = "SPCX.US"
	res := run(ctx, "search_code", map[string]any{"regex": "SPCX", "limit": 8})
	if res.Status != tools.StatusOK {
		fmt.Printf("search_code(SPCX) failed: %s —仍用 %s\n", res.Summary, code)
		return code
	}
	items, _ := res.Data["items"].([]any)
	for _, raw := range items {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		c := strings.TrimSpace(fmt.Sprint(row["code"]))
		if strings.EqualFold(c, code) || strings.EqualFold(c, "SPACEX.US") {
			fmt.Printf("search_code match: %s\n", c)
			break
		}
	}
	return code
}

func runCase(ctx context.Context, run strategydev.ToolRunner, label, code, freq, freqNote string) (ok bool, barCount int) {
	head := fmt.Sprintf("\n======== %s %s (%s) ========\n", label, code, freq)
	if freqNote != "" {
		head = fmt.Sprintf("\n======== %s %s (%s) [%s] ========\n", label, code, freq, freqNote)
	}
	fmt.Print(head)
	probeArgs := map[string]any{
		"code": code, "frequency": strategydev.NormalizeFrequency(freq),
		"months_back": 3,
		"buy_signal":  []any{map[string]any{"index": "RSIThrehold", "type": "signal", "param": map[string]any{"period": 14, "threholdBuy": 30, "threholdSell": 70}}},
		"sell_signal": []any{map[string]any{"index": "RSIThrehold", "type": "signal", "param": map[string]any{"period": 14, "threholdBuy": 30, "threholdSell": 70}}},
	}
	res := run(ctx, "probe_bot_signal_series", probeArgs)
	if res.Status != tools.StatusOK {
		fmt.Printf("probe failed: %s\n", res.Summary)
		return false, 0
	}
	bars, _ := res.Data["bars"].([]any)
	buyM, _ := res.Data["buy_merged"].([]any)
	sellM, _ := res.Data["sell_merged"].([]any)
	fmt.Printf("bars=%d pattern_hits=%d buy_hits=%v\n", len(bars), lenAny(res.Data["pattern_hits"]), res.Data["buy_hits"])
	if len(bars) == 0 {
		return false, 0
	}

	mtf := strategydev.BuildKlinePatternMTFForDiagnose(ctx, run, code, freq, 3, bars)
	patIn := strategydev.NewKlinePatternDisplayInput(freq, bars, res.Data, mtf)
	buyTr, sellTr := strategydev.CollectEpisodeTriggersFromMerged(bars, buyM, sellM)
	strategydev.AttachKlineDisplayToProbeTriggers(buyTr, sellTr, patIn)

	showTriggers(label, bars, buyTr, sellTr)
	return true, len(bars)
}

func showTriggers(label string, bars []any, buyTr, sellTr []strategydev.EpisodeTriggerRecord) {
	show := func(side string, trs []strategydev.EpisodeTriggerRecord) {
		if len(trs) == 0 {
			fmt.Printf("no %s triggers\n", side)
			return
		}
		tr := trs[0]
		fmt.Printf("%s trigger[0] idx=%d time=%s summary=%q bias=%+v\n", side, tr.Index, tr.Time, tr.KlinePatternSummary, tr.KlinePatternBias)
		if tr.KlinePatternThreeBar != nil {
			fmt.Printf("  three_bar anchor_idx=%d note=%q\n", tr.KlinePatternThreeBar.AnchorIndex, tr.KlinePatternThreeBar.WindowNote)
			for _, b := range tr.KlinePatternThreeBar.Bars {
				fmt.Printf("  %s idx=%d omitted=%v close=%.4f patterns=%q bias=%+v\n",
					b.Role, b.Index, b.Omitted, b.Close, b.KlinePatternSummary, b.KlinePatternBias)
			}
		}
		if tr.KlinePatternMTF != nil {
			fmt.Printf("  MTF primary=%+v frames=%d\n", tr.KlinePatternMTF.PrimaryBias, len(tr.KlinePatternMTF.Timeframes))
			for _, tf := range tr.KlinePatternMTF.Timeframes {
				fmt.Printf("    %s aligned=%d bias=%+v has_three_bar=%v\n", tf.Label, tf.AlignedIndex, tf.Bias, tf.ThreeBar != nil)
			}
		}
		walkForwardNote(bars, tr.Index)
		raw, _ := json.MarshalIndent(map[string]any{
			"kline_pattern_bias":      tr.KlinePatternBias,
			"kline_pattern_three_bar": tr.KlinePatternThreeBar,
			"kline_pattern_mtf":       tr.KlinePatternMTF,
		}, "  ", "  ")
		fmt.Printf("  json snippet:\n  %s\n", string(raw))
	}
	show("buy", buyTr)
	if len(sellTr) > 0 {
		show("sell", sellTr)
	}
	_ = label
}

func walkForwardNote(bars []any, triggerIdx int) {
	if triggerIdx < 0 || triggerIdx >= len(bars) {
		return
	}
	startClose := floatAny(fieldAt(bars, triggerIdx, "close"))
	endIdx := triggerIdx + 5
	if endIdx >= len(bars) {
		endIdx = len(bars) - 1
	}
	if endIdx <= triggerIdx {
		return
	}
	endClose := floatAny(fieldAt(bars, endIdx, "close"))
	ret := (endClose - startClose) / startClose * 100
	fmt.Printf("  walk-forward +%d bars: close %.4f → %.4f (%.2f%%, descriptive only)\n",
		endIdx-triggerIdx, startClose, endClose, ret)
}

func fieldAt(bars []any, idx int, key string) any {
	row, _ := bars[idx].(map[string]any)
	if row == nil {
		return nil
	}
	return row[key]
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
