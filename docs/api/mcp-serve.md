# MCP Server (`geegoo mcp serve`)

Expose GeeGooAgent chat/workflow tools to Cursor or other MCP clients over **stdio JSON-RPC**.

## Quick start

```bash
# Chat toolset (default)
geegoo mcp serve --config /path/to/config.json

# Full workflow registry
geegoo mcp serve --toolset workflow --config /path/to/config.json

# Dry-run (no mutating tool side effects)
geegoo mcp serve --dry-run
```

Environment:

- `MCP_TOKEN` or config `mcp_token` — same as GeeGooBot MCP auth when tools call upstream APIs.

## Cursor `mcp.json` example

Add to Cursor **Settings → MCP** or `~/.cursor/mcp.json`:

```json
{
  "mcpServers": {
    "geegoo-agent": {
      "command": "geegoo",
      "args": [
        "mcp",
        "serve",
        "--config",
        "/home/ubuntu/.geegoo/geegoo-agent/config.json",
        "--toolset",
        "chat"
      ],
      "env": {
        "MCP_TOKEN": "your-mcp-token-here"
      }
    }
  }
}
```

On Windows, use the full path to `geegoo.exe` if it is not on `PATH`:

```json
{
  "mcpServers": {
    "geegoo-agent": {
      "command": "D:\\Geegoo\\GeeGooAgent\\geegoo.exe",
      "args": ["mcp", "serve", "--toolset", "chat"],
      "env": {
        "MCP_TOKEN": "your-mcp-token-here"
      }
    }
  }
}
```

## Supported methods (MVP)

| Method | Description |
|--------|-------------|
| `tools/list` | Lists tools from the selected toolset |
| `tools/call` | Executes one tool via the existing `runtime.Executor` |

Not implemented in MVP: resources, prompts, sampling.

## Related

- Headless chat: `geegoo exec -p "分析 00700.HK" --output-format ndjson`
- Runtime events: [runtime-events.md](./runtime-events.md)
