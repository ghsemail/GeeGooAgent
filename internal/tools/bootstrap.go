package tools

import (
	"context"
	"fmt"

	"github.com/ghsemail/GeeGooAgent/internal/clients/mcp"
	"github.com/ghsemail/GeeGooAgent/internal/config"
	"github.com/ghsemail/GeeGooAgent/internal/tools/catalog"
)

// Deps bundles shared dependencies for tool registration.
type Deps struct {
	MCP           *mcp.Client
	WorkspaceRoot string
	ProjectRoot   string
	Working       WorkingLoader
	Search        config.SearchConfig
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
			Handle: func(ctx Context, args map[string]any) Result {
				if ctx.DryRun {
					return Result{
						Status:  StatusDryRun,
						Summary: fmt.Sprintf("dry-run: skipped %s", spec.Name),
						Data:    map[string]any{"tool": spec.Name, "path": spec.Path},
					}
				}
				body := buildHTTPBody(args, spec.MergePayload)
				if spec.RequiresMCPToken {
					body["mcp_token"] = ctx.MCPToken
				}
				var (
					data any
					err  error
				)
				if spec.DirectResponse {
					data, err = deps.MCP.PostDirect(context.Background(), spec.Path, body)
				} else {
					var envelope map[string]any
					envelope, err = deps.MCP.Post(context.Background(), spec.Path, body)
					if err == nil {
						data = envelope["data"]
					}
				}
				if err != nil {
					return Result{Status: StatusError, Summary: err.Error(), ExitCode: 1}
				}
				normalized, summary := normalizeHTTPResponse(spec.Name, data)
				return Result{Status: StatusOK, Summary: summary, Data: normalized}
			},
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
		return v, fmt.Sprintf("%s succeeded", name)
	default:
		return map[string]any{"value": payload}, fmt.Sprintf("%s succeeded", name)
	}
}

// RegisterAll registers HTTP catalog + bespoke tools (~82 total).
func RegisterAll(r *Registry, deps Deps) {
	RegisterHTTPFromCatalog(r, deps)
	RegisterBespokeTools(r, deps)
}

// Names returns sorted registered tool names.
func (r *Registry) Names() []string {
	return r.sortedNames(nil)
}
