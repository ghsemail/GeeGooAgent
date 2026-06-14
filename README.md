# GeeGoo Agent

股票分析专用自托管 Agent 平台（Workflow-first，Phase 1：盘前准备）。

## 快速开始

所有 CLI 命令以 **`geegoo`** 开头（`geegoo-agent` 为兼容别名）。配置与数据默认放在 **`~/.geegoo/`**（可通过 `GEEGOO_HOME` / `GEEGOO_CONFIG` 覆盖）。

### Linux / macOS（推荐）

```bash
curl -fsSL https://raw.githubusercontent.com/ghsemail/GeeGooAgent/main/scripts/install.sh | bash
geegoo setup
geegoo doctor
geegoo chat
```

### 本地开发

```bash
git clone git@github.com:ghsemail/GeeGooAgent.git
cd GeeGooAgent
pip install -e ".[dev]"
geegoo setup
geegoo doctor
geegoo chat
```

更新到最新版：

```bash
geegoo update
geegoo doctor
```

非交互配置示例：

```bash
geegoo setup --non-interactive --provider deepseek --token-key sk-... --mcp-token mcp-... --github-token ghp_...
```

### CLI 命令一览

| 命令 | 说明 |
|------|------|
| `geegoo setup` | 安装依赖（editable）+ 配置 LLM / mcp_token / GeeGoo API |
| `geegoo doctor` | 检查配置、API 连通性与 LLM |
| `geegoo update` | 拉取最新代码并重装（保留 config.json） |
| `geegoo chat` | 交互对话（ReAct + Tool）；`/trace` `/flow` 查看执行轨迹 |
| `geegoo run <skill>` | 运行 workflow（默认 `pre_market`） |
| `geegoo resume --session <id>` | 从 checkpoint 恢复 |

### geegoo chat 示例

```bash
geegoo chat

# 对话中可用斜杠命令：
#   /tools        列出可用 Tool
#   /trace 10     最近 10 步 Tool 调用
#   /flow 15      事件总线（workflow 流水）
#   /run pre_market   跑确定性盘前流水线
#   /dry-run on   切换 dry-run
#   /model        列出可用模型（DeepSeek: v4-flash / v4-pro 等）
#   /model 2      按序号切换模型
#   /verbose on   实时显示思考与 Tool 过程（默认开启，Rich 面板）
#   GEEGOO_CHAT_PLAIN=1  纯文本模式（管道/无 TTY 时自动启用）
#   /exit         退出并保存会话

geegoo chat --session chat-abc123   # 恢复历史会话
```

## 配置说明（`config.example.json`）

默认路径：`~/.geegoo/config.json`（安装脚本或 `geegoo setup` 会自动创建）。

| 字段 | 必填 | 说明 |
|------|------|------|
| `base_url` | ✓ | geegoo mcp（5700）；与 `geegoo_url` 同值 |
| `api_key` | ✓ | geegoo mcp Bearer（env `SK_API_KEY` 可覆盖） |
| `geegoo_url` | ✓ | geegoo mcp（5700） |
| `geegoo_api_key` | ✓ | geegoo mcp Bearer（env `SK_API_KEY`；与 `api_key` 同值） |
| `mcp_token` | ✓ | 用户 MCP token（env `MCP_TOKEN` 或 `--mcp-token` 可覆盖） |
| `output_dir` | ✓ | 工作区；默认 `~/.geegoo/data` |
| `llm.provider` | ✓（实盘） | `openai` / `deepseek` / `minimax` |
| `llm.token_key` | ✓（实盘） | 模型 API Key（env `LLM_TOKEN_KEY` 可覆盖） |
| `llm.model` | | 留空则用各提供商默认模型 |
| `sandbox.allowed_hosts` | ✓ | HTTP 白名单，须包含 API 主机名 |
| `feishu_webhook_url` | | 可选飞书 webhook |

从 TradingBot 同步 Bearer：

```bash
geegoo setup --tradingbot /path/to/TradingBot
```

## 文档

| 文档 | 说明 |
|------|------|
| [docs/architecture/README.md](docs/architecture/README.md) | 架构蓝图 |
| [docs/engineering/requirements.md](docs/engineering/requirements.md) | 工程化要求与验收 |
| [docs/engineering/cursor-workflow.md](docs/engineering/cursor-workflow.md) | 分步开发指南 |
| [PROGRESS.md](PROGRESS.md) | 实现进度（Step 0–15） |

## 进阶：systemd 部署

目标拓扑见 [docs/architecture/cross-cutting/deployment.md](docs/architecture/cross-cutting/deployment.md)。

```bash
# 1. 安装到 /opt/geegoo-agent
sudo useradd -r -m -d /opt/geegoo-agent geegoo-agent || true
sudo -u geegoo-agent git clone <repo> /opt/geegoo-agent
cd /opt/geegoo-agent && sudo -u geegoo-agent python3.11 -m venv venv
sudo -u geegoo-agent venv/bin/pip install -e .

# 2. 配置
sudo mkdir -p /etc/geegoo-agent /var/lib/geegoo-agent/data
sudo cp config.example.json /etc/geegoo-agent/config.json
sudo cp deploy/env.example /etc/geegoo-agent/env
# 编辑 config.json：output_dir → /var/lib/geegoo-agent/data
sudo chmod 600 /etc/geegoo-agent/config.json /etc/geegoo-agent/env
sudo chown root:geegoo-agent /etc/geegoo-agent/config.json /etc/geegoo-agent/env

# 3. systemd
sudo cp deploy/systemd/geegoo-agent-pre-market.service /etc/systemd/system/
sudo cp deploy/systemd/geegoo-agent-pre-market.timer /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable --now geegoo-agent-pre-market.timer

# 手动触发
sudo systemctl start geegoo-agent-pre-market.service
journalctl -u geegoo-agent-pre-market.service -n 30 --no-pager
```

Timer：周一至周五 **08:00 Asia/Shanghai** 触发 `geegoo run pre_market`。

## 真机冒烟

部署后按 [tests/smoke/README.md](tests/smoke/README.md) 执行手动冒烟。

## 从 Hermes 切换

E2E 与冒烟通过后，按 [deploy/hermes-migration-checklist.md](deploy/hermes-migration-checklist.md) 并行验证再切换。

## 开发与测试

```bash
pip install -e ".[dev]"
pytest -q
ruff check src tests
```

## 状态

Phase 1 MVP 盘前 workflow 已实现（Step 0–15）。详见 [PROGRESS.md](PROGRESS.md)。
