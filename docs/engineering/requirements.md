# GeeGoo Agent — 工程化要求文档

> 版本：1.0 | 对齐架构：[architecture/](../architecture/)  
> 实现策略：**Workflow 为主 + LLM 为辅**（Phase 1），非 LangChain，非纯 ReAct 盘前。

---

## 1. 项目目标

| 项 | 要求 |
|----|------|
| **产品目标** | 自托管股票分析 Agent，首个场景：**工作日 8:00 盘前准备**，替代 Hermes `geegoo` cron |
| **架构目标** | 六层 Agent OS（L5→L0），可恢复、可观测、可扩展 |
| **工程目标** | ~2500 行 Phase 0–1 可测代码；dry-run 与 resume；产出与 Hermes 可比 |

---

## 2. 非目标（Phase 1 不做）

- LangChain / LangGraph / LangSmith 作为运行时依赖
- 向量库、embedding、SemanticMemory 实现
- Redis / PostgreSQL / Task Queue
- 任意 Shell Tool、Docker 沙箱、本地 Python 回测
- Bot CRUD（`create_*_bot` / `create_*_reminder`）
- 盘中 / 盘后 / chat 全量（Phase 2+）
- 自动下单

---

## 3. 实现策略（锁定）

### 3.1 Phase 1 运行模式

```text
WorkflowRunner（代码写死 pre_market 步骤顺序）
    ├── 每步：Tool 调用 + Checkpoint + execution-log
    └── LLM 仅用于：周线解析、综合预判、报告正文（窄任务）

ReAct 全循环：Phase 2 再接入（loop.py 可先 stub）
```

**理由**：盘前步骤固定；Workflow 可对照 Hermes、易测、易 resume；降低 Agent 漏步风险。

### 3.2 技术栈（锁定）

| 类别 | 选型 | 禁止 |
|------|------|------|
| 语言 | Python ≥3.11 | — |
| 打包 | `pyproject.toml` + `src/geegoo` | requirements.txt 散落 |
| LLM | `openai` 或 `anthropic` 官方 SDK，经 L1 Gateway | Runtime 直连 SDK |
| HTTP | `httpx` + `tenacity`（3 次，5s） | 自研重试框架 |
| 校验 | `pydantic` v2 | 裸 dict 调 API |
| 日志 | `structlog` 或标准 logging JSONL | — |
| 测试 | `pytest` + `pytest-httpx` | 无测试合并 |
| 模板 | `jinja2`（报告，可选） | — |
| 调度 | **systemd timer**（不写内置 cron） | APScheduler 常驻 |

### 3.3 自研范围上限

| 模块 | 行数预算 | 复杂度上限 |
|------|----------|------------|
| L0 四件套 + Sandbox 简版 | ~400 | 无 async 队列 |
| L1 Gateway 单 provider | ~200 | 仅 chat + tool_calls |
| L2 Clients | ~500 | 薄封装 |
| L2 Tools（MVP 19） | ~700 | 每 Tool ≤50 行 |
| L4 WorkflowRunner | ~400 | 步骤表驱动 |
| L3 Memory | ~300 | 文件 + Pydantic |
| CLI + deploy | ~150 | — |

---

## 4. 仓库结构（必须遵守）

见 [architecture/repo-layout.md](../architecture/repo-layout.md)。Phase 0 脚手架至少包含：

```text
GeeGooAgent/
├── pyproject.toml
├── config.example.json
├── docs/
├── skills/pre_market/
├── rules/
├── references/          # 从 geegoo skill 迁入
├── deploy/
│   ├── geegoo-agent-pre-market.service
│   └── geegoo-agent-pre-market.timer
├── tests/
└── src/geegoo/
    ├── cli.py
    ├── config.py
    ├── infra/
    ├── llm/
    ├── clients/
    ├── tools/
    ├── memory/
    ├── runtime/
    └── supervisor/
```

**依赖方向**：`cli → runtime → {memory, tools, llm}`；`tools → clients`；各层可用 `infra`；**禁止** `infra → tools`。

---

## 5. 配置与密钥

### 5.1 `config.example.json`（提交仓库）

```json
{
  "base_url": "http://118.195.135.97:3120",
  "api_key": "mk-REPLACE",
  "geegoo_url": "http://118.195.135.97:3120",
  "geegoo_api_key": "sk-REPLACE",
  "mcp_token": "REPLACE",
  "output_dir": "./data",
  "llm": {
    "provider": "openai",
    "model": "gpt-4o",
    "api_key_env": "OPENAI_API_KEY"
  },
  "dry_run": false,
  "max_steps": 80
}
```

### 5.2 强制规则

- `mcp_token`、`api_key` **禁止**硬编码进源码或提交 git
- 本地用 `config.json`（gitignore）或环境变量
- GeeGoo HTTP 仅允许配置内 host（Sandbox allowlist）

---

## 6. 分期交付与验收

### Phase 0 — 平台内核（无盘前端到端）

| # | 交付物 | 验收 |
|---|--------|------|
| 0.1 | `pyproject.toml`、`src/geegoo` 包结构 | `pip install -e .` 成功 |
| 0.2 | L0：EventBus、FileStateStore、Checkpoint | 单测：emit 事件、save/load、checkpoint 恢复 |
| 0.3 | L0：Sandbox 简版（路径 + host 白名单） | 单测：越界路径拒绝 |
| 0.4 | L1：Gateway + 单 Provider | 单测：mock SDK 返回 tool_calls |
| 0.5 | L2：`clients/base.py` + `market.py` 三接口 | 集成：mock httpx 调通 checkTradingDay / getReportBotCodes / getCapitalFlow |
| 0.6 | L2：ToolRegistry 骨架 | 可注册、可执行、返回 ToolResult 信封 |
| 0.7 | L4：WorkflowRunner 空壳 + Session 状态机 | `dry-run` 跑 0 步不崩溃 |
| 0.8 | CLI：`geegoo-agent --help` | 子命令占位 |

### Phase 1 — MVP 盘前

| # | 交付物 | 验收 |
|---|--------|------|
| 1.1 | 迁入 `references/`、`rules/`、`skills/pre_market/` | 与 geegoo skill 对齐 |
| 1.2 | `market.py` / `geegoo_bot.py` 全 MVP 端点 | 见 §7 清单 |
| 1.3 | MVP 19 Tool 注册（仅 pre_market manifest） | 单次 run 暴露 ≤25 个 schema |
| 1.4 | `PreMarketWorkflow` 完整步骤 | 对照 `pre-market-workflow.md` |
| 1.5 | LLM 任务：周线解析 + 报告生成 | pydantic 输出校验 |
| 1.6 | `save_local_report` + `create_pre_market_report` | 必填字段 §8；dry-run 跳过 POST |
| 1.7 | `execution-log` + Supervisor 检查清单 | 漏 report 报错 |
| 1.8 | `geegoo-agent resume --session ID` | 从 checkpoint 继续 |
| 1.9 | systemd timer 08:00 + deploy 文档 | 手动 trigger 成功 |
| 1.10 | 端到端 dry-run 测试 | `pytest tests/e2e/test_pre_market_dry_run.py` 绿 |

### Phase 1 成功标准（与 roadmap 一致）

1. 交易日 8:00 自动触发  
2. 每股本地 md + API 入库，`bot_id/bot_name/bot_type` 非空  
3. execution-log 含真实时间戳  
4. Supervisor 可发现漏步；支持 resume  
5. 崩溃后 `geegoo-agent resume` 可继续  
6. 抽检 3 股产出与 Hermes 可比  

---

## 7. MVP Tool 与 API 清单

**注册给 LLM 的 Tool（~19）** — 详见 [tool-catalog.md](../architecture/layers/L2-tools/tool-catalog.md)。

| Tool | 端口 | 备注 |
|------|------|------|
| check_trading_day | 3120 | 非交易日终止 |
| get_report_bot_codes | 3120 | 动态股票列表 |
| fetch_market_news | 本地脚本 | US/CN/HK |
| fetch_stock_news | 本地脚本 | 每股 |
| get_mcp_analysis | 3120 | period 必填；指数 prompt_id 固定 |
| get_stock_daily_reports / list_today_reports | 3120 | 幂等 |
| get_capital_flow | 3120 | period=DAY |
| get_capital_distribution | 3120 | 与上同时调用 |
| get_bot_yesterday_attitude | 3120 | 404→neutral |
| recall_yesterday_summary | 本地 | Episodic |
| read_working_state | Working | — |
| create_pre_market_report | 3120 | §8 必填 |
| save_local_report | 本地 | 工作区内 |
| write_execution_log | 本地 | — |

**禁止封装**：无（getCapitalFlow / getPreMarketReports 已修复，见 clients.md）。

**Scheduled 模式禁止注册**：全部 Bot/Reminder CRUD。

---

## 8. 关键业务校验（实现必须 enforced）

### create_pre_market_report

必填：`mcp_token`, `code`, `stock_name`, `bot_id`, `bot_name`, `bot_type`, `result`, `confidence`, `reason`, `suggestion`, `report`

枚举：`result` ∈ long/short/neutral；`suggestion` ∈ buy/sell/hold；`confidence` ∈ high/medium/low

`bot_id/bot_name/bot_type` **必须**来自 `get_report_bot_codes`，禁止空字符串。

### attitude → result

| attitude | result |
|----------|--------|
| bullish | long |
| bearish | short |
| neutral | neutral |

### get_mcp_analysis

- `period`：`hourly`（指数/小时），`weekly`（个股周线）；**禁止** `hour`
- `name`：股票中文名，非 prompt 名

---

## 9. 测试要求

> **完整规范**见 [testing-standards.md](./testing-standards.md)（每 Step 必交付用例表）与 [coding-standards.md](./coding-standards.md)。

### 9.1 铁律

- 每 Step（0–15）必须完成 testing-standards §5 对应测试交付物  
- `pytest -q` 全绿方可进入下一步  
- 禁止单测/集成测访问真实 GeeGoo API 与真实 LLM（Step 15 手动冒烟除外）  
- 修 bug 必须先写失败用例  

### 9.2 金字塔

| 层级 | 比例 | 内容 |
|------|------|------|
| 单元 | 多 | StateStore、Checkpoint、Sandbox、Pydantic 模型、映射函数 |
| 集成 | 中 | httpx mock GeeGoo API；Workflow 段式 dry-run |
| E2E | 少 | 全链路 dry-run（LLM mock） |

### 9.3 每个 Phase 合并门槛

- `pytest -q` 全绿 + `ruff check src tests`  
- Step 5 起覆盖率建议 ≥60%；Step 14 起 ≥70%  
- 本 Phase 验收表逐项勾选  
- **不**在无 E2E dry-run 通过的情况下接真 LLM  

### 9.4 推荐测试文件

见 [testing-standards.md §3–§5](./testing-standards.md)。

---

## 10. 可观测性

| 项 | 要求 |
|----|------|
| execution-log | `{output_dir}/{date}/execution-log.md`，每步一行 |
| 技术日志 | JSONL：`{output_dir}/{date}/run-{session_id}.jsonl` |
| 事件 | MVP 至少：`RunStarted`, `ToolCalled`, `ToolCompleted`, `CheckpointSaved`, `RunFinished`, `RunFailed` |
| dry-run | 日志标明 `DRY_RUN`；写接口跳过 HTTP |

---

## 11. 部署

| 项 | 要求 |
|----|------|
| 目标环境 | Linux（与现 Hermes 同机或新 VM） |
| 触发 | `deploy/geegoo-agent-pre-market.timer` → `geegoo-agent run pre_market` |
| 切换 | 盘前 E2E 通过后再禁用 Hermes cron |
| 配置路径 | 服务器 `config.json` 权限 `600` |

---

## 12. 质量门禁（Code Review 检查项）

- [ ] 未引入 LangChain/LangGraph 依赖  
- [ ] Runtime 未直连 openai/anthropic（经 Gateway）  
- [ ] 新 Tool 已加入 manifest 白名单；未向 Scheduled 暴露 Bot CRUD  
- [ ] API 调用经 Client，带 Authorization + Content-Type  
- [ ] 敏感字段不进 git  
- [ ] 单测覆盖 happy path + 关键失败路径（404 attitude、非交易日）  
- [ ] 单文件未超过 400 行（超过则拆模块）  

---

## 13. 风险与缓解

| 风险 | 缓解 |
|------|------|
| 自研 Agent 漏步 | WorkflowRunner 写死顺序；Supervisor 跑后检查 |
| Tool 爆炸 | 仅 manifest 注册；catalog 文档不等于运行时注册 |
| API 字段漏传 | Pydantic 模型 + 发 API 前校验 |
| LLM 幻觉 | 结构化字段用 pydantic；prose 仅 report 段落 |
| 服务不稳定 | tenacity 3 次；timeout ≥60s |
| 与 Hermes 不一致 | 逐步骤 diff；保留 references 原文 |

---

## 14. 参考文档索引

| 主题 | 路径 |
|------|------|
| 架构总览 | [architecture/00-overview.md](../architecture/00-overview.md) |
| 盘前步骤 | 迁入 `references/pre-market-workflow.md` |
| API 路由 | [architecture/domains/geegoo-api-routing.md](../architecture/domains/geegoo-api-routing.md) |
| Clients | [architecture/layers/L2-tools/clients.md](../architecture/layers/L2-tools/clients.md) |
| Sandbox | [architecture/L0-infrastructure/sandbox.md](../architecture/L0-infrastructure/sandbox.md) |
| 外部依赖 | [architecture/layers/L3-memory/README.md](../architecture/layers/L3-memory/README.md) |

---

## 15. 文档变更

| 版本 | 日期 | 说明 |
|------|------|------|
| 1.0 | 2026-06-05 | 首版：Workflow-first Phase 1 |
