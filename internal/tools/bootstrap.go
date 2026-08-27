package tools

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/clients/weknora"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/memory/episodic"
	"github.com/ghsemail/GeeGooAgent/internal/memory/facts"
	"github.com/ghsemail/GeeGooAgent/internal/memory/procedural"
	"github.com/ghsemail/GeeGooAgent/internal/memory/scoped"
	"github.com/ghsemail/GeeGooAgent/internal/memport"
	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

// Deps bundles shared dependencies for tool registration.
type Deps struct {
	HTTP             HTTPBackends
	WorkspaceRoot    string
	ProjectRoot      string
	Working          WorkingLoader
	Search           config.SearchConfig
	FeishuWebhookURL string
	Memory           memport.Port
	Facts            *facts.PostgresStore
	Episodic         *episodic.PostgresStore
	Preferences      *scoped.PreferencesStore
	SkillLoader      *procedural.Loader
	Home             string
	WeKnora          *weknora.Client
	// Delegate runs delegate_task sub-agent turns (optional; wired by app).
	Delegate TaskDelegator
}

// TaskDelegator is implemented by agent.SubAgent for delegate_task registration.
type TaskDelegator interface {
	DelegateTask(ctx Context, task, background string, maxSteps int) Result
	DelegateTasks(ctx Context, tasks []BatchDelegateTask) Result
}

// BatchDelegateTask is one item for delegate_tasks.
type BatchDelegateTask struct {
	Task       string
	Background string
	MaxSteps   int
}

// WorkingLoader loads working memory for meta tools.
type WorkingLoader interface {
	Load(sessionID string) (map[string]any, error)
}

// RegisterHTTPFromCatalog adds generic MCP forwarding tools.
func RegisterHTTPFromCatalog(r *Registry, deps Deps) {
	for _, spec := range catalog.AllHTTP() {
		spec := spec
		r.Register(Tool{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
			Handle: ApprovalGate(spec.Name, func(ctx Context, args map[string]any) Result {
				if ctx.DryRun {
					return Result{
						Status:  StatusDryRun,
						Summary: fmt.Sprintf("dry-run: skipped %s", spec.Name),
						Data:    map[string]any{"tool": spec.Name, "path": spec.Path},
					}
				}
				body := buildHTTPBody(args, spec.MergePayload)
				if spec.Name == "run_strategy_backtest" {
					NormalizeBacktestStrategyLink(body)
				}
				if uid := strings.TrimSpace(ctx.UserID); uid != "" {
					switch spec.Name {
					case "run_strategy_backtest", "list_strategy_backtest_logs":
						body["user_id"] = uid
					}
					if spec.Name == "run_strategy_backtest" {
						if _, ok := body["source"]; !ok {
							body["source"] = "agent"
						}
					}
				}
				if catalog.NeedsMCPToken(spec.Name) {
					if strings.TrimSpace(ctx.MCPToken) == "" {
						return Result{
							Status: StatusError, Summary: "缺少 mcp_token：请运行 geegoo setup 配置",
							ExitCode: 1,
						}
					}
					body["mcp_token"] = ctx.MCPToken
				}
				started := time.Now()
				client := deps.HTTP.ForTool(spec.Name)
				var last Result
				for attempt := 0; attempt < 2; attempt++ {
					var (
						data     any
						envelope map[string]any
						err      error
					)
					if spec.DirectResponse {
						data, err = client.PostDirect(ctx.GoContext(), spec.Path, body)
					} else {
						envelope, err = client.Post(ctx.GoContext(), spec.Path, body)
						if err == nil {
							data = envelope["data"]
						}
					}
					if err != nil && deps.HTTP.HasMCPFallback(spec.Name) && mcp.ShouldAnalyzeFallback(err) {
						fallback := deps.HTTP.MCP
						if spec.DirectResponse {
							data, err = fallback.PostDirect(ctx.GoContext(), spec.Path, body)
						} else {
							envelope, err = fallback.Post(ctx.GoContext(), spec.Path, body)
							if err == nil {
								data = envelope["data"]
							}
						}
					}
					if err != nil {
						if attempt == 0 && shouldRetryHTTPEmpty(spec.Name) {
							if waitRetry(ctx.GoContext()) {
								continue
							}
						}
						return Result{Status: StatusError, Summary: enrichHTTPError(spec.Name, err), ExitCode: 1,
							Meta: MetaFromEnvelope(nil, started)}
					}
					data = compactHTTPPayload(ctx, spec.Name, data)
					normalized, summary := normalizeHTTPResponse(spec.Name, data)
					if spec.Name == "generate_grid_strategy" {
						summary = appendStrategyFollowUp(summary, "grid", normalized)
					}
					if spec.Name == "generate_dca_strategy" {
						summary = appendStrategyFollowUp(summary, "dca", normalized)
					}
					meta := MetaFromEnvelope(envelope, started)
					if status, note, _ := ClassifyHTTPPayload(spec.Name, normalized, envelope); status != StatusOK {
						last = Result{Status: status, Summary: note, Data: normalized, Meta: meta}
						if attempt == 0 && shouldRetryHTTPEmpty(spec.Name) {
							if waitRetry(ctx.GoContext()) {
								continue
							}
						}
						return last
					}
					return Result{Status: StatusOK, Summary: summary, Data: normalized, Meta: meta}
				}
				return last
			}),
		})
	}
}

func buildHTTPBody(args map[string]any, mergePayload bool) map[string]any {
	if !mergePayload {
		out := map[string]any{}
		for k, v := range args {
			out[k] = v
		}
		return out
	}
	out := map[string]any{}
	if payload, ok := args["payload"].(map[string]any); ok {
		for k, v := range payload {
			out[k] = v
		}
	}
	for k, v := range args {
		if k != "payload" {
			out[k] = v
		}
	}
	return out
}

func enrichHTTPError(toolName string, err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	if !strings.Contains(msg, "502") {
		return msg
	}
	switch toolName {
	case "generate_dca_strategy":
		return msg + "；analyze-api :3230 业务层异常：确认 signal_id 来自 get_signal_combinations/get_index_signals，并检查 GeeGooSignal analyze-api 日志与 LLM 配置"
	case "generate_grid_strategy":
		return msg + "；analyze-api :3230 业务层异常：检查 GeeGooSignal analyze-api 日志与 LLM 配置"
	default:
		return msg
	}
}

func normalizeHTTPResponse(name string, payload any) (map[string]any, string) {
	switch v := payload.(type) {
	case []any:
		return map[string]any{"items": v, "count": len(v)}, summarizeCatalogList(name, v)
	case map[string]any:
		if price, ok := v["price"]; ok {
			return v, fmt.Sprintf("%s: price=%v", name, price)
		}
		if finalValue, ok := v["finalValue"]; ok {
			if profitRate, ok := v["profit_rate"]; ok {
				return v, fmt.Sprintf("%s: finalValue=%v profit_rate=%v", name, finalValue, profitRate)
			}
		}
		switch name {
		case "probe_bot_signal_series":
			if bars, ok := v["bars"].([]any); ok {
				return v, fmt.Sprintf("probe_bot_signal_series: %d bars, buy_hits=%d sell_hits=%d",
					len(bars), countMergedSignals(v["buy_merged"], 1), countMergedSignals(v["sell_merged"], -1))
			}
		case "probe_bot_signal":
			buy := nestedInt(v, "buy_signal", "signal")
			sell := nestedInt(v, "sell_signal", "signal")
			return v, fmt.Sprintf("probe_bot_signal: buy=%d sell=%d close=%v", buy, sell, v["close"])
		case "list_strategy_backtest_logs":
			if items, ok := v["items"].([]any); ok {
				msg := fmt.Sprintf("list_strategy_backtest_logs: %d record(s)", len(items))
				if len(items) > 0 {
					if row, ok := items[0].(map[string]any); ok {
						profitRate := nestedAny(row, "result", "profit_rate")
						msg += fmt.Sprintf("; latest code=%v strategy=%v profit_rate=%v log_id=%v",
							row["code"], row["strategy_label"], profitRate, row["log_id"])
					}
				}
				return v, msg
			}
		case "run_strategy_backtest":
			return v, fmt.Sprintf("run_strategy_backtest: log_id=%v code=%v profit_rate=%v final_value=%v",
				v["log_id"], v["code"], v["profit_rate"], v["final_value"])
		case "get_strategy_backtest_log":
			if run, ok := v["run"].(map[string]any); ok {
				profitRate := nestedAny(run, "result", "profit_rate")
				trades := 0
				if t, ok := v["trades"].([]any); ok {
					trades = len(t)
				}
				return v, fmt.Sprintf("get_strategy_backtest_log: code=%v profit_rate=%v trades=%d",
					run["code"], profitRate, trades)
			}
		case "get_indicator_series":
			if values, ok := v["values"].([]any); ok {
				return v, fmt.Sprintf("get_indicator_series: %d value(s) index=%v", len(values), v["index"])
			}
		}
		return v, fmt.Sprintf("%s succeeded", name)
	default:
		return map[string]any{"value": payload}, fmt.Sprintf("%s succeeded", name)
	}
}

func countMergedSignals(raw any, target int) int {
	arr, ok := raw.([]any)
	if !ok {
		return 0
	}
	n := 0
	for _, item := range arr {
		switch v := item.(type) {
		case float64:
			if int(v) == target {
				n++
			}
		case int:
			if v == target {
				n++
			}
		}
	}
	return n
}

func nestedInt(m map[string]any, keys ...string) int {
	v := nestedAny(m, keys...)
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func nestedAny(m map[string]any, keys ...string) any {
	var cur any = m
	for _, key := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return nil
		}
		cur = obj[key]
	}
	return cur
}

func appendStrategyFollowUp(summary, strategyType string, data map[string]any) string {
	if data == nil {
		return summary
	}
	switch strategyType {
	case "grid":
		if param, ok := data["param"].(map[string]any); ok && len(param) > 0 {
			return summary + "；可 loopback_strategy(type=grid, grid_param=param) 或 create_grid_bot(grid=param)"
		}
	case "dca":
		if signal, ok := data["signal"].(map[string]any); ok {
			if buy, ok := signal["buy_signal"].([]any); ok && len(buy) > 0 {
				return summary + "；可 loopback_strategy(type=dca, signal=signal.buy_signal, sl_tp 由 dynamicParam/fixedParam 组装) 或 create_dca_bot"
			}
		}
	}
	return summary
}

// RegisterAll registers HTTP catalog + bespoke tools (~82 total).
// Registrars can be extended via AddRegistrar (Go-side toolset self-registration).
func RegisterAll(r *Registry, deps Deps) {
	for _, reg := range registrarsSnapshot() {
		reg(r, deps)
	}
}

// Registrar registers one batch of tools (catalog, bespoke, or a future toolset).
type Registrar func(*Registry, Deps)

var (
	registrarMu sync.RWMutex
	registrars  []Registrar
)

// AddRegistrar appends a tool registrar (for tests or optional tool packs).
func AddRegistrar(reg Registrar) {
	if reg == nil {
		return
	}
	registrarMu.Lock()
	registrars = append(registrars, reg)
	registrarMu.Unlock()
}

func registrarsSnapshot() []Registrar {
	registrarMu.RLock()
	defer registrarMu.RUnlock()
	out := make([]Registrar, len(registrars))
	copy(out, registrars)
	return out
}

// Names returns sorted registered tool names.
func (r *Registry) Names() []string {
	return r.sortedNames(nil)
}
