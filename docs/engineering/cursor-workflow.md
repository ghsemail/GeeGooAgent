# Cursor Agent 分步开发指南

> 如何用 Cursor 智能体**高效率、低返工**完成 GeeGoo Agent。  
> 前提：已读 [requirements.md](./requirements.md)、[coding-standards.md](./coding-standards.md)、[testing-standards.md](./testing-standards.md)。

---

## 1. 总原则

| 原则             | 说明                                                                                                |
| -------------- | ------------------------------------------------------------------------------------------------- |
| **一步一会话**      | 每个 Step 开一个 Cursor Agent 会话（或干净上下文），避免上下文污染                                                       |
| **先读后写**       | 每步开头 `@` 引用本步「必读文件」，禁止_agent 凭记忆编 API                                                             |
| **先测后并**       | 每步交付必须含 [testing-standards.md §5](./testing-standards.md#5-分-step-测试交付物强制) 全部用例；`pytest` 未绿不进入下一步 |
| **小 PR 心态**    | 每步 diff 控制在 ~300–500 行；过大则拆 Step                                                                  |
| **dry-run 优先** | Phase 1 完成前，默认 `dry_run=true`                                                                     |
| **不扩 scope**   | Agent 请求里写清「本步不做什么」                                                                               |

**不推荐**：一个会话里「帮我把整个 Agent 写完」——极易漏测、漏字段、乱引框架。

---

## 2. 开发顺序总览

```text
Step 0  脚手架 + pyproject
Step 1  L0 四件套（EventBus / StateStore / Checkpoint）
Step 2  L0 Sandbox 简版 + Secrets + config
Step 3  L1 Gateway（单 Provider mock 测）
Step 4  L2 clients/base + market.py（3 个 API + 集成测）
Step 5  L2 ToolRegistry + 首批 3 Tool
Step 6  L3 WorkingMemory（PreMarketWorking 模型）
Step 7  L4 WorkflowRunner 空壳 + CLI run/resume
Step 8  迁入 references / rules / skills/pre_market
Step 9  L2 补全 MVP Clients + 19 Tool
Step 10 PreMarketWorkflow 阶段 A（指数+新闻）
Step 11 PreMarketWorkflow 阶段 B（个股+报告）
Step 12 LLM 任务（解析+报告 pydantic）
Step 13 Supervisor + execution-log
Step 14 E2E dry-run 测试
Step 15 deploy systemd + 真机冒烟
```

预估：**15 个 Agent 会话**，每会话 30–90 分钟。

---

## 3. 会话设置建议

### 3.1 Cursor Rules（建议写入 `.cursor/rules/geegoo-agent.md`）

```markdown
- 实现依据：docs/engineering/requirements.md、coding-standards.md、testing-standards.md
- 禁止：LangChain、LangGraph、向量库、Shell Tool
- Phase 1：WorkflowRunner 写死顺序，不用全 ReAct
- GeeGoo API 字段以 docs/architecture/domains/ 与 clients.md 为准
- 每步必须完成 testing-standards.md §5 该 Step 行全部测试；HTTP 用 pytest-httpx mock
- 禁止访问真实 GeeGoo API / 真实 LLM（Step 15 手动冒烟除外）
- 只改本步列出的文件；不要重构无关代码
- 遵循 coding-standards.md：Pydantic、类型注解、单文件 ≤400 行
- 交付末尾列出：新增测试文件、用例数、已跑 pytest 结果
```

### 3.2 模式选择

| 阶段         | 模式                                 |
| ---------- | ---------------------------------- |
| Step 0–2   | **Agent**（写代码）                     |
| 架构疑问       | **Ask** 或读 `docs/architecture`     |
| 大范围方案变更    | **Plan** 先对齐再 Agent                |
| Step 14–15 | **Agent** + 你在终端跑 pytest / systemd |

### 3.3 分支策略

```bash
main          # 稳定
develop       # 集成分支（可选）
feat/step-N-* # 每步一个分支，合并前 pytest 绿
```

每完成一个 Step：`git commit`（用户明确要求时），message 如 `feat(step-4): market client with httpx tests`。

---

## 4. 分步指令模板（复制即用）

> 将 `Step N` 整段复制到 **新 Cursor Agent 会话** 作为首条消息。  
> 替换 `@文件` 为 Cursor 的 @ 引用。

**每条 Step 已含测试要求**；完整用例表见 [testing-standards.md §5](./testing-standards.md#5-分-step-测试交付物强制)。

**通用测试附录**（可追加到任一步）：

```markdown
## 测试要求（必做）
@docs/engineering/testing-standards.md
@docs/engineering/coding-standards.md
- 完成 testing-standards §5 本 Step 行全部用例
- 禁止真实 GeeGoo / LLM 请求
- 交付后执行：pytest -q && ruff check src tests
- 回复中列出：测试文件路径、用例数、覆盖场景
```

---

### Step 0 — 脚手架

```markdown
@docs/engineering/requirements.md
@docs/architecture/repo-layout.md

实现 Step 0：项目脚手架。
交付：pyproject.toml（Python 3.11+，依赖 openai/anthropic/httpx/tenacity/pydantic/pytest/pytest-httpx）、
src/geegoo/__init__.py、config.example.json、.gitignore（含 config.json、data/）、
空包 infra/llm/clients/tools/memory/runtime/supervisor、tests/conftest.py。
本步不做业务逻辑。验收：pip install -e ".[dev]" && pytest 空通过。
```

---

### Step 1 — L0 四件套

```markdown
@docs/engineering/requirements.md
@docs/architecture/layers/L0-infrastructure/event-bus.md
@docs/architecture/layers/L0-infrastructure/state-store.md
@docs/architecture/layers/L0-infrastructure/checkpoint.md

实现 Step 1：L0 EventBus（InProcess 同步）、FileStateStore、Checkpoint。
代码：src/geegoo/infra/events.py、state_store.py、checkpoint.py。
单测：tests/unit/test_state_store.py、test_checkpoint.py、test_event_bus.py。
本步不做 Sandbox/Gateway。验收：pytest tests/unit/ -q 全绿。
```

---

### Step 2 — Sandbox + Config

```markdown
@docs/architecture/layers/L0-infrastructure/sandbox.md
@docs/architecture/layers/L0-infrastructure/secrets.md
@docs/engineering/requirements.md

实现 Step 2：SandboxManager 简版（工作区路径校验、HTTP host allowlist）、
config.py 加载 config.json、Secrets 从 env 读 LLM key。
单测：越界路径 ../etc 拒绝；非 allowlist host 拒绝。
本步不实现完整六层 Sandbox。验收：pytest 相关单测绿。
```

---

### Step 3 — L1 Gateway

```markdown
@docs/architecture/layers/L1-model-gateway/gateway.md
@docs/architecture/layers/L1-model-gateway/providers.md

实现 Step 3：ModelGateway + OpenAIProvider（或 AnthropicProvider 二选一）。
统一 ToolCall 输出；tenacity 3 次重试。CostManager 可 stub。
单测：mock SDK，断言 tool_calls 解析。本步不接 Workflow。
```

---

### Step 4 — Market Client（3 API）

```markdown
@docs/architecture/layers/L2-tools/clients.md
@docs/architecture/domains/geegoo-api-routing.md
@C:\Users\ghsemail\.cursor\skills\geegoo\docs\geegoo-mcp/market/trading-data.md

实现 Step 4：clients/base.py（Bearer、重试、timeout 60s）、
clients/market.py：check_trading_day、get_report_bot_codes、get_capital_flow。
集成测 pytest-httpx mock 3120。不要硬编码 mcp_token。
本步不做 Tool 封装。
```

---

### Step 5 — ToolRegistry + 3 Tool

```markdown
@docs/architecture/layers/L2-tools/registry.md
@docs/architecture/layers/L2-tools/sandbox-integration.md

实现 Step 5：ToolRegistry、ToolResult 信封、SandboxManager 集成；
tools/perceive.py：check_trading_day、get_report_bot_codes、write_execution_log。
Executor 骨架可放在 runtime/executor.py（仅调用 registry）。
单测：mock client 执行 check_trading_day。
```

---

### Step 6 — WorkingMemory

```markdown
@docs/architecture/layers/L3-memory/working-memory.md
@docs/engineering/requirements.md §8

实现 Step 6：PreMarketWorking（pydantic）— 字段含 phase、indices_done、
stocks、per_stock 子结构；memory/working.py 读写 StateStore。
本步不做 Episodic/Semantic。
```

---

### Step 7 — WorkflowRunner 空壳 + CLI

```markdown
@docs/architecture/layers/L4-runtime/workflow-engine.md
@docs/architecture/layers/L4-runtime/state-machine.md

实现 Step 7：WorkflowRunner 接受步骤列表、驱动 Session 状态机、
每步 checkpoint；cli.py 子命令 run pre_market、resume --session、--dry-run。
先注册 1 个假步骤通过端到冒烟。本步不接 LLM。
```

---

### Step 8 — 迁入 Skill 资产

```markdown
@C:\Users\ghsemail\.cursor\skills\geegoo\references\
@C:\Users\ghsemail\.cursor\skills\geegoo\SKILL.md
@docs/architecture/domains/geegoo-skill-mapping.md

实现 Step 8：复制并整理 references/（pre-market-workflow、template、api-routing）、
rules/（attitude-mapping、api-routing、report-format）、
skills/pre_market/SKILL.md + manifest.yaml（tools 白名单、步骤列表）。
本步以文件迁入为主，不改 runtime 逻辑。
```

---

### Step 9 — 补全 Clients + 19 Tool

```markdown
@docs/architecture/layers/L2-tools/tool-catalog.md
@docs/architecture/layers/L2-tools/clients.md
@skills/pre_market/manifest.yaml

实现 Step 9：补全 market.py、geegoo_bot.py 剩余 MVP 端点；
按 manifest 注册全部 MVP Tool（约 19 个）；新闻脚本放入 skills/bundled/ 或 scripts/。
每个 Tool pydantic 入参；create_pre_market_report 发 API 前校验 §8。
集成测 mock 关键 API。
```

---

### Step 10 — 阶段 A Workflow

```markdown
@references/pre-market-workflow.md
@skills/pre_market/manifest.yaml

实现 Step 10：PreMarketWorkflow 阶段 A—
check_trading_day → get_report_bot_codes → 5 指数 get_mcp_analysis（可 asyncio 并行）
→ fetch_market_news。每步写 execution-log + checkpoint。
dry-run 下 get_mcp_analysis 可用 fixture。单测 test_workflow_phase_a.py。
本步不做个股循环。
```

---

### Step 11 — 阶段 B Workflow

```markdown
@references/pre-market-workflow.md
@docs/engineering/requirements.md §8

实现 Step 11：阶段 B 每股循环—
新闻、get_capital_flow、get_capital_distribution、weekly get_mcp_analysis、
get_bot_yesterday_attitude、save_local_report、create_pre_market_report。
404 attitude → neutral；幂等 list_today_reports。
```

---

### Step 12 — LLM 任务

```markdown
@references/pre-market-template.md
@docs/architecture/layers/L1-model-gateway/gateway.md

实现 Step 12：runtime/llm_tasks.py—
parse_weekly_analysis(markdown)->结构化字段；
synthesize_report(context)->报告 pydantic；
经 Gateway 调用，失败重试 3 次。
单测：mock LLM 返回固定 JSON/markdown。本步不把全循环交给 ReAct。
```

---

### Step 13 — Supervisor

```markdown
@docs/architecture/cross-cutting/supervisor.md

实现 Step 13：supervisor/pre_market.py—
跑后检查：每股是否有 md、create 是否成功、必填字段；
输出摘要到 execution-log。与 resume 联调。
```

---

### Step 14 — E2E dry-run

```markdown
@docs/engineering/requirements.md §9

实现 Step 14：tests/e2e/test_pre_market_dry_run.py—
mock 全部 HTTP + LLM，跑完整 PreMarketWorkflow；
断言 working 终态、checkpoint 存在、日志条数。
修到 pytest 全绿。本步禁止接真 API。
```

---

### Step 15 — 部署与冒烟

```markdown
@docs/architecture/cross-cutting/deployment.md
@docs/engineering/requirements.md §11

实现 Step 15：deploy/systemd unit+timer、README 部署节、
config.example.json 说明。给出真机冒烟命令（非交易日可测 checkTradingDay 终止路径）。
本步不禁用 Hermes，只写切换检查清单。
```

---

## 5. 每步验收清单（你来做）

详见 [testing-standards.md §8](./testing-standards.md#8-每步验收命令复制执行)。

```bash
pip install -e ".[dev]"
pytest -q
pytest --cov=geegoo --cov-report=term-missing   # Step 5 起
ruff check src tests
ruff format --check src tests
geegoo-agent run pre_market --dry-run   # Step 7 起
```

| 检查                   | 通过标准                   |
| -------------------- | ---------------------- |
| testing-standards §5 | 本 Step 行全部打勾           |
| pytest               | 0 failed               |
| 覆盖率                  | Step 5≥60%，Step 14≥70% |
| 文件数                  | 单文件 ≤400 行；仅本步路径       |
| 依赖                   | 无 langchain            |
| Agent 交付说明           | 含测试文件列表 + 用例数          |

---

## 6. Agent 跑偏时怎么办

| 症状           | 处理                                                    |
| ------------ | ----------------------------------------------------- |
| 引入 LangChain | 回滚该提交；会话强调 requirements §3.2                          |
| 一次改 20+ 文件   | 中断；拆成两个 Step 重做                                       |
| API 字段猜错     | `@` 指向 `geegoo-mcp/market/trading-data.md` / `clients.md` 重做该 Client |
| 纯 ReAct 盘前   | 强调 WorkflowRunner；ReAct 仅 Phase 2                     |
| 无测试          | 拒绝合并；补「本步验收：pytest xxx 绿」                             |
| 87 Tool 全注册  | 仅 manifest 中的 19 个                                    |

**恢复口令**（新会话首条）：

```markdown
当前进度：Step N 已完成。请只读 @docs/engineering/cursor-workflow.md Step N+1，
不要修改 Step N 已验收的文件，除非修 bug。
```

---

## 7. 推荐日程（单人 + Cursor）

| 天   | Step  | 产出                          |
| --- | ----- | --------------------------- |
| D1  | 0–2   | 脚手架 + L0 + Sandbox          |
| D2  | 3–5   | Gateway + Client + Registry |
| D3  | 6–8   | Memory + CLI + skill 迁入     |
| D4  | 9–10  | 全 Tool + 阶段 A               |
| D5  | 11–12 | 阶段 B + LLM                  |
| D6  | 13–14 | Supervisor + E2E            |
| D7  | 15    | 部署 + 真机对比 Hermes            |

---

## 8. 与架构文档的关系

```text
architecture/  = 做什么、为什么（设计）
engineering/   = 怎么做、怎么验（执行）
cursor-workflow.md = 谁（Cursor Agent）按什么顺序做
```

架构变更时：**先改 architecture**，再改 requirements 验收项，最后改本指南 Step 描述。

---

## 9. 下一步

1. 新建 Cursor 会话
2. 复制 **Step 0** 指令
3. 验收 pytest 后进入 Step 1

需要我代劳 Step 0 时，直接说「执行 Step 0」。