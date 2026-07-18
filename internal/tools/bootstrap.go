package tools

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ghsemail/GeeGooAgent/internal/config"
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
	// Delegate runs delegate_task sub-agent turns (optional; wired by app).
	Delegate TaskDelegator
}

// TaskDelegator is implemented by agent.SubAgent for delegate_task registration.
type TaskDelegator interface {
	DelegateTask(ctx Context, task, background string, maxSteps int) Result
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
					if err != nil && deps.HTTP.HasMCPFallback(spec.Name) {
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
						return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1,
							Meta: MetaFromEnvelope(nil, started)}
					}
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

func normalizeHTTPResponse(name string, payload any) (map[string]any, string) {
	switch v := payload.(type) {
	case []any:
		return map[string]any{"items": v, "count": len(v)}, fmt.Sprintf("%s: %d item(s)", name, len(v))
	case map[string]any:
		if price, ok := v["price"]; ok {
			return v, fmt.Sprintf("%s: price=%v", name, price)
		}
		if finalValue, ok := v["finalValue"]; ok {
			if profitRate, ok := v["profit_rate"]; ok {
				return v, fmt.Sprintf("%s: finalValue=%v profit_rate=%v", name, finalValue, profitRate)
			}
		}
		return v, fmt.Sprintf("%s succeeded", name)
	default:
		return map[string]any{"value": payload}, fmt.Sprintf("%s succeeded", name)
	}
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
