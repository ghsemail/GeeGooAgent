# GeeGoo Agent

GeeGoo Agent is a Go-native workflow agent for market analysis and pre-market report automation.

The project now builds and runs as a single Go binary named `geegoo`. Configuration and local runtime data default to `~/.geegoo/`, and can be overridden with `GEEGOO_HOME` and `GEEGOO_CONFIG`.

## Quick Start

### Linux / macOS

```bash
curl -fsSL https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | bash
geegoo setup
geegoo doctor
geegoo chat
```

### Local Development

```bash
git clone git@github.com:ghsemail/GeeGooAgent.git
cd GeeGooAgent
go build ./cmd/geegoo
./geegoo setup
./geegoo doctor
./geegoo chat
```

Update to the latest version:

```bash
geegoo update
geegoo doctor
```

## CLI

| Command | Description |
| --- | --- |
| `geegoo setup` | Write a default config for the Go runtime. |
| `geegoo doctor` | Check config, MCP connectivity, and LLM readiness. |
| `geegoo update` | Pull the latest code and rebuild the Go binary. |
| `geegoo chat` | Start the interactive ReAct + tool chat loop. |
| `geegoo run pre_market` | Run the deterministic pre-market workflow. |
| `geegoo resume --session <id>` | Resume a checkpointed pre-market session. |

## Runtime Ports

The current GeeGoo service layout uses 3xxx endpoints:

| Service | Port | Purpose |
| --- | ---: | --- |
| MCP | `3120` | Main MCP API and workflow/report endpoints. |
| Signal | `3210` | Signal and strategy generation service. |
| Analyze | `3230` | Analysis service. |
| Data | `3300` | Market data service. |

`geegoo setup` writes these defaults into `config.json`. The application still warns when legacy 5xxx ports are detected in existing configs.

## Configuration

Default config path:

```text
~/.geegoo/config.json
```

Important fields:

| Field | Required | Description |
| --- | --- | --- |
| `base_url` | yes | MCP endpoint, default `http://127.0.0.1:3120`. |
| `api_key` | yes | MCP bearer key. Can be provided with `SK_API_KEY`. |
| `geegoo_url` | yes | Same MCP endpoint used by tool clients. |
| `geegoo_api_key` | yes | Same bearer key used by tool clients. |
| `mcp_token` | yes | User MCP token. Can be provided with `MCP_TOKEN`. |
| `output_dir` | yes | Working data directory, default `~/.geegoo/data`. |
| `llm.provider` | live runs | `openai`, `deepseek`, or `minimax`. |
| `llm.token_key` | live runs | Model API key. Can be provided with `LLM_TOKEN_KEY`. |
| `sandbox.allowed_hosts` | yes | HTTP allowlist for tool calls. |

## Systemd Deployment

```bash
sudo useradd -r -m -d /opt/geegoo-agent geegoo-agent || true
sudo -u geegoo-agent git clone <repo> /opt/geegoo-agent
cd /opt/geegoo-agent
sudo -u geegoo-agent go build -o geegoo ./cmd/geegoo

sudo mkdir -p /etc/geegoo-agent /var/lib/geegoo-agent/data
sudo -u geegoo-agent ./geegoo setup --config /etc/geegoo-agent/config.json --force
sudo cp deploy/env.example /etc/geegoo-agent/env
sudo chmod 600 /etc/geegoo-agent/config.json /etc/geegoo-agent/env
sudo chown root:geegoo-agent /etc/geegoo-agent/config.json /etc/geegoo-agent/env

sudo cp deploy/systemd/geegoo-agent-pre-market.service /etc/systemd/system/
sudo cp deploy/systemd/geegoo-agent-pre-market.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now geegoo-agent-pre-market.timer
```

Manual trigger:

```bash
sudo systemctl start geegoo-agent-pre-market.service
journalctl -u geegoo-agent-pre-market.service -n 30 --no-pager
```

The timer runs `geegoo run pre_market` at 08:00 Asia/Shanghai on weekdays.

## Verification

```bash
go test ./...
go build ./cmd/geegoo
```
