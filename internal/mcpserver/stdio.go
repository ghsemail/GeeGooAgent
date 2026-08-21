package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ghsemail/GeeGooAgent/internal/app"
	"github.com/ghsemail/GeeGooAgent/internal/tools"
)

// ServeStdio runs a minimal MCP tools server over stdin/stdout (JSON-RPC 2.0).
func ServeStdio(application *app.App, toolNames []string) error {
	if application == nil || application.Registry == nil {
		return fmt.Errorf("mcpserver: app not configured")
	}
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	enc := json.NewEncoder(os.Stdout)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		resp := handleRequest(context.Background(), application, toolNames, req)
		if req.ID != nil {
			_ = enc.Encode(resp)
		}
	}
	return sc.Err()
}

type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResponse struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func handleRequest(ctx context.Context, application *app.App, toolNames []string, req rpcRequest) rpcResponse {
	resp := rpcResponse{JSONRPC: "2.0", ID: req.ID}
	switch req.Method {
	case "initialize":
		resp.Result = map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "geegoo-agent", "version": "1.0"},
		}
	case "notifications/initialized", "initialized":
		// notification, no response body when id nil
	case "tools/list":
		schemas := application.Registry.Schemas(toolNames)
		toolsOut := make([]map[string]any, 0, len(schemas))
		for _, s := range schemas {
			toolsOut = append(toolsOut, map[string]any{
				"name":        s.Name,
				"description": s.Description,
				"inputSchema": s.Parameters,
			})
		}
		resp.Result = map[string]any{"tools": toolsOut}
	case "tools/call":
		var params struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			resp.Error = &rpcError{Code: -32602, Message: "invalid params"}
			return resp
		}
		mcpToken := strings.TrimSpace(os.Getenv("MCP_TOKEN"))
		if mcpToken == "" && application.Config != nil {
			mcpToken = application.Config.UserMCPToken
		}
		toolCtx := application.ToolContextWithContext(ctx, "mcp-serve")
		toolCtx.MCPToken = mcpToken
		toolCtx.Approved = true
		result := application.Registry.Execute(tools.CallRequest{
			Name:      params.Name,
			Arguments: params.Arguments,
		}, toolCtx)
		text := result.Summary
		if result.Data != nil {
			if b, err := json.Marshal(result.Data); err == nil {
				text = string(b)
			}
		}
		resp.Result = map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
			"isError": result.Status == tools.StatusError,
		}
	default:
		resp.Error = &rpcError{Code: -32601, Message: "method not found: " + req.Method}
	}
	return resp
}

// DiscardReader drains optional stderr helper input.
func DiscardReader(r io.Reader) {
	io.Copy(io.Discard, r)
}
