package tools

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

const (
	hourlyPricePromptID  = "663e5ac904f98788e502fab7"
	hourlySignalPromptID = "6644cbbdf729b97ea8b59275"
	hourlyKlinePromptID  = "66475a36fc8d11278ed561ae"
)

type hourlyBundleSlot struct {
	key      string
	promptID string
}

var hourlyBundleSlots = []hourlyBundleSlot{
	{key: "price_analysis", promptID: hourlyPricePromptID},
	{key: "signal_analysis", promptID: hourlySignalPromptID},
	{key: "kline_analysis", promptID: hourlyKlinePromptID},
}

func registerHourlyAnalysisBundle(r *Registry, deps Deps) {
	r.Register(Tool{
		Name:        "get_hourly_analysis_bundle",
		Description: "并行执行个股小时级 MCP 分析（价格/信号/K 线三模板）。盘后 postmarket_stock 与高频 intraday 使用，替代三次串行 get_mcp_analysis。",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name": map[string]any{
					"type":        "string",
					"description": "股票名称",
				},
				"code": map[string]any{
					"type":        "string",
					"description": "股票代码，如 00700.HK",
				},
				"language": map[string]any{
					"type":        "string",
					"description": "输出语言 cn/en/hk，默认 cn",
				},
			},
			"required": []any{"name", "code"},
		},
		Handle: func(ctx Context, args map[string]any) Result {
			name := strArg(args, "name", "")
			code := strArg(args, "code", "")
			language := strArg(args, "language", "cn")
			if name == "" || code == "" {
				return Result{
					Status: StatusError, ExitCode: 1,
					Summary: "get_hourly_analysis_bundle 需要 name 与 code",
				}
			}
			if ctx.DryRun {
				return okDryRun("get_hourly_analysis_bundle", map[string]any{
					"code": code, "period": "hourly",
					"price_analysis":  fmt.Sprintf("[dry-run] hourly price for %s", name),
					"signal_analysis": fmt.Sprintf("[dry-run] hourly signal for %s", name),
					"kline_analysis":  fmt.Sprintf("[dry-run] hourly kline for %s", name),
				})
			}
			data, partial, err := runHourlyAnalysisBundle(ctx.GoContext(), deps, ctx.MCPToken, name, code, language)
			if err != nil {
				return errResult(err)
			}
			status := StatusOK
			summary := fmt.Sprintf(
				"并行 MCP hourly 分析完成 %s（price/signal/kline）。答复用户时请按 SOUL 将各 analysis 改写为标题+列表，勿照抄 |表格| 或 ---。",
				code,
			)
			if len(partial) > 0 {
				summary += " 部分子项失败: " + strings.Join(partial, "; ")
			}
			if statusNote, note, _ := ClassifyHTTPPayload("get_hourly_analysis_bundle", data, nil); statusNote != StatusOK {
				if hasAnyHourlyAnalysis(data) {
					status = StatusOK
					summary = note + " " + summary
				} else {
					return Result{Status: statusNote, Summary: note, Data: data}
				}
			}
			return Result{Status: status, Summary: summary, Data: data}
		},
	})
}

func runHourlyAnalysisBundle(
	ctx context.Context,
	deps Deps,
	mcpToken, name, code, language string,
) (map[string]any, []string, error) {
	type slotResult struct {
		key  string
		text string
		err  error
	}
	ch := make(chan slotResult, len(hourlyBundleSlots))
	var wg sync.WaitGroup
	for _, slot := range hourlyBundleSlots {
		wg.Add(1)
		go func(slot hourlyBundleSlot) {
			defer wg.Done()
			res, err := getMCPAnalysisResilient(ctx, deps.HTTP.MCP, mcpToken, name, code, slot.promptID, "hourly", language)
			if err != nil {
				ch <- slotResult{key: slot.key, err: err}
				return
			}
			text := ""
			if res != nil {
				text = strings.TrimSpace(res.AnalysisResult)
			}
			ch <- slotResult{key: slot.key, text: text}
		}(slot)
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	data := map[string]any{"code": code, "period": "hourly"}
	var partial []string
	okCount := 0
	for r := range ch {
		if r.err != nil {
			partial = append(partial, fmt.Sprintf("%s: %v", r.key, r.err))
			continue
		}
		data[r.key] = r.text
		if r.text != "" {
			okCount++
		}
	}
	if okCount == 0 {
		if len(partial) > 0 {
			return data, partial, fmt.Errorf("hourly bundle failed: %s", strings.Join(partial, "; "))
		}
		return data, partial, fmt.Errorf("hourly bundle returned no analysis for %s", code)
	}
	return data, partial, nil
}

func hasAnyHourlyAnalysis(data map[string]any) bool {
	for _, key := range []string{"price_analysis", "signal_analysis", "kline_analysis"} {
		if s, _ := data[key].(string); strings.TrimSpace(s) != "" {
			return true
		}
	}
	return false
}
