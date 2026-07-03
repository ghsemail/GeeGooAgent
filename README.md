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
| `geegoo run <skill>` | Run a skill workflow (e.g. `pre_market`). See `geegoo skills list`. |
| `geegoo resume --session <id>` | Resume a checkpointed workflow (idempotent by step key). |
| `geegoo migrate [--dry-run]` | Migrate legacy file-based chat sessions to SQLite. |
| `geegoo skills list` | List registered skills. |
| `geegoo scheduler run` | Start the in-process cron scheduler (long-running). |
| `geegoo scheduler list` | Show scheduled jobs and last verdicts. |
| `geegoo verify --codes <list> [--date <D>]` | Cutover field-completeness verification of pre-market reports. |

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
geegoo verify --codes 00700.HK,000001.SZ --date <YYYY-MM-DD>
```

## Architecture (Hermes Parity)

GeeGooAgent is benchmarked against the Hermes Agent architecture and reorganized into a platform-agnostic Go core with entry points (CLI, HTTP runtime) at the edges.

```text
internal/
  agent/        platform-agnostic core: Agent.Run(ctx, session, input)
  provider/     (named llm/ today) gateway + OpenAI-compatible providers
  tools/        registry, contract, approval, catalog, bespoke
  session/      (named chatsession/ today) SQLite + FTS5 session store
  memory/       working memory + EvidenceStore (SQLite)
  workflow/     runner, pre_market steps, supervisor, errors
  skills/       manifest-driven skill registry
  scheduler/    in-process cron + supervisor-driven retry
  report/       LLM evidence-only synthesis (result/confidence stay rule-based)
  verify/       cutover field-completeness checks
  infra/        SQLite handle + schema, state, events, guard
  cli/          chatrepl + chatui + commands
  clients/mcp/  GeeGooBot MCP client
  search/       free web search (DuckDuckGo)
  config/ doctor/ auth/ httpserver/
```

Key capabilities delivered in P1–P8:

- **SQLite foundation**: chat sessions, evidence records, working state, checkpoints, execution events in one WAL-enabled DB with FTS5; `geegoo migrate` from legacy JSON.
- **Prompt stability**: system message stays byte-identical across turns; tool activity injected as dynamic user-side context → DeepSeek prefix cache friendly.
- **Interruptible**: `context.Context` threaded through provider/gateway/tools/loop; Ctrl+C aborts in-flight LLM + tool calls.
- **Supervisor + idempotent resume**: post-run verdict (pass/recoverable/terminal); resume skips by step key, not step number; recoverable errors auto-retry once.
- **Skill registry**: `geegoo run <skill>` dispatches via manifest; intraday/post_market placeholders registered.
- **LLM report synthesis**: LLM writes reason/suggestion/summary strictly from evidence; result/confidence stay rule-based so it cannot flip a decision.
- **Tool contracts**: `Result.Meta`, empty-success detection (code=100 but empty → Skip), approval gate for mutating tools, fixture replay tests.
- **In-process scheduler**: cron-driven skill execution with exponential-backoff retry on non-pass verdicts.
- **Cutover verification**: `geegoo verify` quantifies bot_id/bot_name/bot_type non-empty rate, enum validity, reason length, evidence_refs presence.

Full roadmap and Hermes comparison: [`deploy/hermes-parity-roadmap.md`](deploy/hermes-parity-roadmap.md). Cutover runbook: [`deploy/hermes-migration-checklist.md`](deploy/hermes-migration-checklist.md).
