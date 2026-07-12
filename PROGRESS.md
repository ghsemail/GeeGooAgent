# GeeGoo Agent 实现进度

> 对照 [docs/engineering/cursor-workflow.md](docs/engineering/cursor-workflow.md)

## 任务表

- **Step 0** — 脚手架 pyproject + 包结构 + conftest
- **Step 1** — L0 EventBus / StateStore / Checkpoint
- **Step 2** — Sandbox 简版 + Config + Secrets
- **Step 3** — L1 Model Gateway + Provider
- **Step 4** — Market Client 三 API + 集成测
- **Step 5** — ToolRegistry + 首批 3 Tool
- **Step 6** — WorkingMemory (PreMarketWorking)
- **Step 7** — WorkflowRunner 空壳 + CLI
- **Step 8** — 迁入 references / rules / skills/pre_market
- **Step 9** — 补全 Clients + MVP 19 Tool
- **Step 10** — PreMarketWorkflow 阶段 A
- [x] **Step 11** — PreMarketWorkflow 阶段 B
- **Step 12** — LLM 任务 (解析+报告)
- **Step 13** — Supervisor
- **Step 14** — E2E dry-run 全链路测试
- **Step 15** — systemd 部署 + 冒烟文档

## 当前

- **进行中**：Step 12
- **最近验收**：Step 11 — `pytest` 131 passed，`ruff` 通过

---

## Step 完成记录

### Step 0 — 脚手架 ✅

**交付物**

| 类别     | 文件                                                                           |
| ------ | ---------------------------------------------------------------------------- |
| 打包     | `pyproject.toml`                                                             |
| 配置样例   | `config.example.json`                                                        |
| 忽略规则   | `.gitignore`                                                                 |
| 包入口    | `src/geegoo/__init__.py`                                                 |
| CLI 占位 | `src/geegoo/cli.py`                                                      |
| 空包     | `infra/`, `llm/`, `clients/`, `tools/`, `memory/`, `runtime/`, `supervisor/` |
| 测试     | `tests/conftest.py`, `tests/fixtures/geegoo/.gitkeep`                          |

**测试**（testing-standards §5）

| 文件                            | 用例数 | 场景                                 |
| ----------------------------- | --- | ---------------------------------- |
| `tests/unit/test_scaffold.py` | 4   | 版本号、CLI help、run/resume 参数、fixture |

**验收命令**

```bash
pip install -e ".[dev]"
pytest -q    # 4 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 1 — L0 四件套 ✅

**交付物**

| 类别         | 文件                                    | 说明                                                     |
| ---------- | ------------------------------------- | ------------------------------------------------------ |
| 异常基类       | `src/geegoo/exceptions.py`        | `GeeGooAgentError` 及 State/Checkpoint 子类                 |
| EventBus   | `src/geegoo/infra/events.py`      | `InProcessEventBus`：同步分发、history、handler 异常隔离          |
| StateStore | `src/geegoo/infra/state_store.py` | `FileStateStore`：JSON 落盘、key 前缀列表、非法 key 拒绝            |
| Checkpoint | `src/geegoo/infra/checkpoint.py`  | `CheckpointManager`：save/load_latest/list/load_working |
| 包导出        | `src/geegoo/infra/__init__.py`    | 导出上述公开类型                                               |

**测试**（testing-standards §5，要求各 ≥3 用例）

| 文件                               | 用例数 | 覆盖场景                                                     |
| -------------------------------- | --- | -------------------------------------------------------- |
| `tests/unit/test_event_bus.py`   | 3   | handler 调用、history 记录、异常不阻断其他 handler                    |
| `tests/unit/test_state_store.py` | 6   | 往返读写、缺 key、前缀列表、删除、非法 key、损坏 JSON                        |
| `tests/unit/test_checkpoint.py`  | 5   | save/load_latest、未知 session、list 排序、最新 step、working 缺失抛错 |

**未做（属 Step 2+）**

- Sandbox / Config / Secrets
- Scheduler adapter（仍用后续 deploy/systemd）

**验收命令**

```bash
pytest -q    # 18 passed（含 Step 0 共 18）
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 2 — Sandbox + Config + Secrets ✅

**交付物**

| 类别  | 文件                                | 说明                                                |
| --- | --------------------------------- | ------------------------------------------------- |
| 配置  | `src/geegoo/config.py`        | `AppConfig` + `load_config()`（Pydantic）           |
| 密钥  | `src/geegoo/infra/secrets.py` | `ConfigSecrets`：env 优先于 config                    |
| 沙箱  | `src/geegoo/infra/sandbox.py` | `WorkspaceGuard`、`NetworkPolicy`、`SandboxManager` |
| 异常  | `exceptions.py`                   | 新增 `SandboxError`                                 |
| 导出  | `infra/__init__.py`               | 导出 Secrets / Sandbox 类型                           |

**测试**（testing-standards §5，各 ≥4 用例）

| 文件                           | 用例数 | 覆盖场景                                             |
| ---------------------------- | --- | ------------------------------------------------ |
| `tests/unit/test_config.py`  | 5   | 加载成功、缺文件、坏 JSON、缺字段、默认值                          |
| `tests/unit/test_sandbox.py` | 6   | 合法相对路径、`../` 越界、绝对路径、host 白名单/拒绝、Manager         |
| `tests/unit/test_secrets.py` | 5   | 读 config、env 覆盖、LLM key、placeholder 拒绝、masked 脱敏 |

**未做（属 Step 3+）**

- Resource Guard（timeout/大小限制）完整实现
- `clients/base.py` HTTP 集成 allowlist

**验收命令**

```bash
pytest -q    # 34 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 3 — L1 Model Gateway + Provider ✅

**交付物**

| 类别        | 文件                          | 说明                                                           |
| --------- | --------------------------- | ------------------------------------------------------------ |
| 类型        | `llm/types.py`              | `Message`、`ToolSchema`、`ToolCall`、`LLMResponse`、`TokenUsage` |
| 协议        | `llm/base.py`               | `LLMProvider` Protocol                                       |
| 成本        | `llm/cost.py`               | `CostManager` 记录 token                                       |
| OpenAI    | `llm/openai_provider.py`    | `tool_calls` 解析；`create_fn` 可注入（测试用）                         |
| Anthropic | `llm/anthropic_provider.py` | `tool_use` 解析；system 消息分离                                    |
| 网关        | `llm/gateway.py`            | 重试 3 次 + 可选 fallback + Cost 记录                               |
| 异常        | `exceptions.py`             | `ModelGatewayError`                                          |
| 导出        | `llm/__init__.py`           | 公开 API                                                       |

**测试**（testing-standards §5）

| 文件                                      | 用例数 | 覆盖场景                        |
| --------------------------------------- | --- | --------------------------- |
| `tests/unit/test_openai_provider.py`    | 3   | tool_calls 解析、文本响应、tools 传参 |
| `tests/unit/test_anthropic_provider.py` | 2   | tool_use 解析、system 提取       |
| `tests/unit/test_gateway.py`            | 5   | 成功、重试、fallback、全失败、cost 记录  |
| `tests/unit/test_cost.py`               | 2   | session 汇总、未知 session       |

**未做（属 Step 4+）**

- metrics.json 落盘（CostManager 仅内存）
- USD 单价估算

**验收命令**

```bash
pytest -q    # 46 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 4 — Market Client 三 API ✅

**交付物**

| 类别       | 文件                           | 说明                                                          |
| -------- | ---------------------------- | ----------------------------------------------------------- |
| 基类       | `clients/base.py`            | httpx POST、Bearer、重试 3×、NetworkPolicy、业务码校验                 |
| 市场       | `clients/market.py`          | `check_trading_day`、`get_report_bot_codes`、`get_capital_flow` |
| 模型       | 同上                           | Pydantic：`TradingDayData`、`UserBotCode`、`CapitalFlowItem`   |
| 异常       | `exceptions.py`              | `ClientError`（api_code / http_status）                       |
| Fixtures | `tests/fixtures/geegoo/*.json` | 三个 API 成功响应样例                                               |

**测试**（testing-standards §5，≥6 用例）

| 文件                                        | 用例数 | 覆盖场景                                              |
| ----------------------------------------- | --- | ------------------------------------------------- |
| `tests/integration/test_market_client.py` | 7   | 3 API happy；code 102；401 API Key；500 重试失败；host 拒绝 |

**验收命令**

```bash
pytest -q    # 53 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 5 — ToolRegistry + 首批 3 Tool ✅

**交付物**

| 类别  | 文件                         | 说明                                           |
| --- | -------------------------- | -------------------------------------------- |
| 类型  | `tools/types.py`           | `ToolContext`、`ToolResult`、`ToolCallRequest` |
| 基类  | `tools/base.py`            | `BaseTool` + Pydantic 校验 + `to_schema()`     |
| 注册  | `tools/registry.py`        | 注册/过滤/执行；Scheduled 禁 Bot mutation            |
| 沙箱  | `infra/sandbox_manager.py` | Tool 执行包装 + summary 截断                       |
| 感知  | `tools/perceive.py`        | `check_trading_day`、`get_report_bot_codes`     |
| 元操作 | `tools/meta.py`            | `write_execution_log`                        |
| 引导  | `tools/bootstrap.py`       | `register_mvp_tools()`                       |
| 执行器 | `runtime/executor.py`      | 委派 Registry + EventBus                       |

**测试**（testing-standards §5）

| 文件                                  | 用例数 | 覆盖场景                                |
| ----------------------------------- | --- | ----------------------------------- |
| `tests/unit/test_tool_registry.py`  | 5   | 注册列表、未知 tool、schema 过滤、事件、scheduled |
| `tests/unit/test_tools_perceive.py` | 5   | 交易日、dry-run、bot 列表、写日志、Executor     |

**验收命令**

```bash
pytest -q    # 63 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 6 — WorkingMemory ✅

**交付物**

| 类别  | 文件                   | 说明                                                             |
| --- | -------------------- | -------------------------------------------------------------- |
| 模型  | `memory/models.py`   | `PreMarketWorking`、`BotStock`、`StockWorkspace`、`MarketContext` |
| 存储  | `memory/working.py`  | `WorkingMemoryStore`：create/load/save/apply/summary            |
| 导出  | `memory/__init__.py` | 公开类型                                                           |

**apply 逻辑（MVP）**

- `check_trading_day` → `is_trading_day`；真→`phase_a`，假→`done`
- `get_report_bot_codes` → `bot_codes` + 初始化 `stocks`；→`phase_b`
- `write_execution_log` → `artifacts.execution_log`

**测试**（testing-standards §5，≥4 用例）

| 文件                                  | 用例数 | 覆盖场景                             |
| ----------------------------------- | --- | -------------------------------- |
| `tests/unit/test_working_memory.py` | 5   | 往返、交易日、bot 初始化、summary、非交易日 done |

**验收命令**

```bash
pytest -q    # 68 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 7 — WorkflowRunner 空壳 + CLI ✅

**交付物**

| 类别       | 文件                    | 说明                                                             |
| -------- | --------------------- | -------------------------------------------------------------- |
| 会话       | `runtime/session.py`  | `Session`、`SessionManager`（create/load/save）                   |
| 工作流      | `runtime/workflow.py` | `WorkflowRunner`、`PRE_MARKET_STUB_STEPS`（3 步 stub）、`RunResult` |
| 组装       | `runtime/app.py`      | `GeeGooApp.from_config`；`run_skill` / `resume_session`           |
| CLI      | `cli.py`              | `run pre_market [--dry-run]`、`resume --session`                |
| Fixtures | `tests/conftest.py`   | `sample_config`、`sample_config_file`                           |

**MVP 工作流（stub）**

1. `check_trading_day` → 非交易日短路完成
2. `get_report_bot_codes` → 写入 working
3. `write_execution_log` → 落盘执行日志

**测试**（testing-standards §5，≥5 用例）

| 文件                                   | 用例数 | 覆盖场景                                               |
| ------------------------------------ | --- | -------------------------------------------------- |
| `tests/unit/test_workflow_runner.py` | 5   | Session；全流程；非交易日短路；未知 tool 失败；resume               |
| `tests/unit/test_cli.py`             | 7   | help；dry-run；缺 config；resume 缺失/成功；不支持 skill；失败退出码 |

**验收命令**

```bash
pytest -q    # 80 passed
ruff check src tests
geegoo-agent run pre_market --dry-run --config config.json
```

**完成日期**：2026-06-05

---

### Step 8 — 迁入 Skill 资产 ✅

**交付物**

| 类别      | 路径                                              | 说明                                                   |
| ------- | ----------------------------------------------- | ---------------------------------------------------- |
| 工作流     | `skills/pre_market/workflow.md`                 | 自 `pre-market-workflow.md` 迁入，路径改为 `skills/bundled/` |
| 模板      | `skills/pre_market/template.md`                 | 自 `pre-market-template.md` 迁入，注明 RSI/MACD 限制         |
| Skill   | `skills/pre_market/SKILL.md`                    | 盘前 Skill Pack 说明                                     |
| 清单      | `skills/pre_market/manifest.yaml`               | 16 Tool 白名单 + 3 LLM 任务 + 步骤表                         |
| Rules   | `rules/api-routing.md`                          | 3120 路由（含 2026-05-20 修复说明）                      |
| Rules   | `rules/attitude-mapping.md`                     | attitude→result、404 处理                               |
| Rules   | `rules/report-format.md`                        | create_pre_market_report 必填字段、九章结构                   |
| Bundled | `skills/bundled/finance-news/`                  | `fetch_news.py`                                      |
| Bundled | `skills/bundled/eastmoney-news/`                | `search.py`                                          |
| Bundled | `skills/bundled/free-stock-global-quotes-news/` | `news.py`（备选）                                        |
| 加载器     | `runtime/skill_loader.py`                       | 读取/校验 manifest 与资产路径                                 |
| 依赖      | `pyproject.toml`                                | 新增 `pyyaml`                                          |

**测试**（testing-standards §5，≥2 用例）

| 文件                                | 用例数 | 覆盖场景                                                 |
| --------------------------------- | --- | ---------------------------------------------------- |
| `tests/unit/test_skill_loader.py` | 5   | manifest 加载；MVP tools 齐全；资产路径存在；未知 skill；workflow 阶段 |

**验收命令**

```bash
pytest -q    # 85 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 9 — 补全 Clients + MVP 19 Tool ✅

**交付物**

| 类别     | 文件                     | 说明                                                                                                                    |
| ------ | ---------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Client | `clients/geegoo_bot.py`  | `get_mcp_analysis`、`get_stock_daily_reports`（3120）                                                                    |
| Client | `clients/market.py`    | 扩展：`get_capital_distribution`、`get_bot_yesterday_attitude`（404→neutral）、`get_mcp_analysis`、`create_pre_market_report` |
| 校验     | `tools/schemas.py`     | `PreMarketReportCreate` 必填字段 + 枚举                                                                                     |
| 映射     | `tools/mappings.py`    | `attitude_to_result`、资金分布格式化                                                                                          |
| 分析     | `tools/analyze.py`     | 4 Tool：mcp / capital_flow / capital_distribution / bot_attitude                                                       |
| 报告     | `tools/act_reports.py` | create / save_local / get_stock_daily / list_today                                                                    |
| 新闻     | `tools/news.py`        | fetch_market_news、fetch_stock_news（bundled 脚本）                                                                        |
| 决策     | `tools/decide.py`      | recall_yesterday_summary、read_working_state                                                                           |
| 通知     | `tools/act_notify.py`  | send_feishu_summary（webhook 可选）                                                                                       |
| 注册     | `tools/bootstrap.py`   | 按 manifest 注册 16 Tool                                                                                                 |
| 上下文    | `tools/types.py`       | ToolContext 扩展 geegoo_bot / working_store / project_root                                                                |
| 应用     | `runtime/app.py`       | 注入 GeeGooBotClient + 全量 Tool                                                                                            |

**测试**（testing-standards §5，≥20 累计新增）

| 文件                                          | 用例数 | 覆盖场景                                                |
| ------------------------------------------- | --- | --------------------------------------------------- |
| `tests/integration/test_market_client.py`   | +5  | capital_distribution、attitude、404、mcp、create_report |
| `tests/integration/test_geegoo_bot_client.py` | 2   | mcp_analysis、daily_reports                          |
| `tests/unit/test_tools_analyze.py`          | 5   | 4 分析 Tool + dry_run                                 |
| `tests/unit/test_tools_act_reports.py`      | 5   | create 校验、save_local、查询、幂等                          |
| `tests/unit/test_tools_news.py`             | 3   | 市场/个股新闻 mock 脚本                                     |
| `tests/unit/test_tools_decide.py`           | 2   | recall、read_working_state                           |
| `tests/unit/test_report_schema.py`          | 4   | PreMarketReportCreate 校验                            |
| `tests/unit/test_mappings.py`               | 4   | attitude 映射、资金分布格式                                  |
| `tests/unit/test_bootstrap.py`              | 2   | manifest 与注册表对齐                                     |

**验收命令**

```bash
pytest -q    # 117 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 11 — PreMarketWorkflow 阶段 B ✅

**交付物**

| 类别 | 文件 | 说明 |
|------|------|------|
| 步骤表 | `runtime/pre_market_workflow.py` | `PRE_MARKET_PER_STOCK_STEPS`（9 步/股） |
| 报告桩 | `runtime/pre_market_report.py` | 阶段 B stub 报告与 `create_pre_market_report` 参数 |
| Runner | `runtime/workflow.py` | `per_stock_steps` 每股循环；幂等 skip；`phase_b`→`done` |
| Working | `memory/models.py`、`memory/working.py` | 每股字段更新；`list_today_reports` 幂等 |
| 应用 | `runtime/app.py` | 全链路 A+B |

**阶段 B 流程（每股）**

`list_today_reports`（已存在→skip）→ `fetch_stock_news` → `get_capital_flow` → `get_capital_distribution` → weekly `get_mcp_analysis` → `get_bot_yesterday_attitude` → `save_local_report` → `create_pre_market_report`

**测试**（testing-standards §5，≥6 用例）

| 文件 | 用例数 | 覆盖场景 |
|------|--------|----------|
| `tests/unit/test_workflow_phase_b.py` | 7 | dry-run 单股；live happy；404→neutral；幂等 skip；双股；bot 字段校验 |

**验收命令**

```bash
pytest -q    # 131 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 12 — LLM 任务（解析 + 报告）✅

**交付物**

| 类别 | 文件 | 说明 |
|------|------|------|
| LLM 任务 | `runtime/llm_tasks.py` | `parse_weekly_analysis`、`synthesize_pre_market_report`、`enrich_stock_with_llm` |
| 报告构建 | `runtime/pre_market_report.py` | 有 `synthesis` 时用 LLM 结果，否则 fallback stub |
| 工作流 | `runtime/workflow.py` | `get_bot_yesterday_attitude` 后非 dry-run 调用 LLM  enrichment |
| 应用 | `runtime/app.py` | 可选构建 `ModelGateway` 注入 `ToolContext` |
| 类型 | `memory/models.py`、`tools/types.py` | `weekly_parsed` / `synthesis` 字段；`llm_gateway` |
| 异常 | `exceptions.py` | `LLMTaskError`（JSON / pydantic 校验失败） |

**LLM 流程（每股，非 dry-run）**

`get_bot_yesterday_attitude` → `parse_weekly_analysis` → `synthesize_pre_market_report` → `save_local_report` / `create_pre_market_report`

**测试**（testing-standards §5，≥4 用例）

| 文件 | 用例数 | 覆盖场景 |
|------|--------|----------|
| `tests/unit/test_llm_tasks.py` | 6 | mock 固定 JSON；周线解析；报告合成；pydantic 失败；enrich 写 workspace；报告 args |
| `tests/fixtures/llm/` | 2 | `weekly_parsed_ok.json`、`synthesis_ok.json` |

**验收命令**

```bash
pytest -q    # 137 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 13 — Supervisor + execution-log ✅

**交付物**

| 类别 | 文件 | 说明 |
|------|------|------|
| 检查清单 | `skills/pre_market/supervisor_checks.yaml` | 5 项：phase、本地 md、API report_id、bot 字段、payload 校验 |
| 检查器 | `supervisor/checks.py` | `file_exists`、`stocks_status`、`api_response`、`execution_log_contains` |
| 引擎 | `supervisor/engine.py` | 加载 YAML、逐项 verify、返回 `SupervisorResult` |
| 入口 | `supervisor/pre_market.py` | `run_pre_market_supervisor` |
| 应用 | `runtime/app.py` | workflow 完成后跑 supervisor；摘要写入 execution-log；`run_skill` 含阶段 B |

**检查项（交易日）**

每股 `reported` → 本地 `{code}-premarket.md` 存在、`report_id` 非空、bot 字段齐全、`PreMarketReportCreate` 校验通过；`phase=done`

**测试**（testing-standards §5，≥3 用例）

| 文件 | 用例数 | 覆盖场景 |
|------|--------|----------|
| `tests/unit/test_supervisor.py` | 5 | 全通过；缺 md 失败；缺 report_id 失败；非交易日跳过；YAML 加载 |

**验收命令**

```bash
pytest -q    # 142 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 14 — E2E dry-run 全链路 ✅

**交付物**

| 类别 | 文件 | 说明 |
|------|------|------|
| E2E 测试 | `tests/e2e/test_pre_market_dry_run.py` | CLI `--dry-run` 全链路 + resume 幂等 |
| dry-run 数据 | `runtime/pre_market_constants.py`、`tools/perceive.py` | `get_report_bot_codes` 返回 sample bot，支撑阶段 B |

**E2E 断言清单（testing-standards §4.3）**

- [x] `execution-log.md` 含阶段 A/B 与 supervisor 步骤名
- [x] `checkpoints/` 最终 step = 20（11 阶段 A + 9 阶段 B）
- [x] 每股有 `{code}-premarket.md`
- [x] dry-run 无 HTTP 请求（含 `createPreMarketReport`）
- [x] `working` 终态 `phase=done`；supervisor 通过

**测试**

| 文件 | 用例数 | 覆盖场景 |
|------|--------|----------|
| `tests/e2e/test_pre_market_dry_run.py` | 2 | 全链路 dry-run；completed 后 resume 幂等 |

**验收命令**

```bash
pytest -q    # 145 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

### Step 15 — 部署与冒烟 ✅

**交付物**

| 类别 | 文件 | 说明 |
|------|------|------|
| systemd | `deploy/systemd/geegoo-agent-pre-market.service` | oneshot：`geegoo-agent run pre_market` |
| systemd | `deploy/systemd/geegoo-agent-pre-market.timer` | Mon–Fri 08:00 `Asia/Shanghai` |
| 环境模板 | `deploy/env.example` | `/etc/geegoo-agent/env` 示例（LLM + 密钥覆盖） |
| 切换清单 | `deploy/hermes-migration-checklist.md` | 并行验证 → 切换 → 回滚（不自动禁用 Hermes） |
| 冒烟文档 | `tests/smoke/README.md` | 真机手动冒烟表（含非交易日短路） |
| 配置 | `config.example.json` | 补全 `feishu_webhook_url`；字段说明见 README |
| 文档 | `README.md` | 快速开始、配置表、部署、冒烟、切换指引 |

**验收**

- 文档含 systemd 安装命令与非交易日 `checkTradingDay` 冒烟路径
- `pytest -q` 仍全绿（冒烟不自动化）

**验收命令**

```bash
pytest -q    # 145 passed
ruff check src tests
```

**完成日期**：2026-06-05

---

## Phase 1 完成

Step 0–15 全部交付。真机冒烟与 Hermes 切换见 `tests/smoke/README.md`、`deploy/hermes-migration-checklist.md`。

## 已知问题

（无）