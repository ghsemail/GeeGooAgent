# Tools 与 Skills

GeeGooAgent 的能力由两层协作提供：

- **Skill**：「跑什么任务」— 工作流边界、步骤顺序、报告模板、质检规则
- **Tool**：「能调什么 API」— LLM 通过 function calling 触发的原子能力

对照 Hermes：`toolsets.py` + `skills/` 目录；GeeGoo 用 `internal/tools/toolset.go` + `internal/skills` + `skills/` 资源目录。

---

## 概念关系

```text
┌─────────────────────────────────────────────────────────────┐
│  Skill（任务包）                                              │
│  skills/pre_market/manifest.yaml + workflow.md + template   │
│       │                                                      │
│       ├── 声明 tools[] 白名单（文档 + 运行时过滤参考）          │
│       ├── PhaseA / PerStock 步骤（Go: workflow/*.go）        │
│       └── supervisor_checks.yaml（跑后验收）                  │
└───────────────────────────┬─────────────────────────────────┘
                            │ 步骤内调用
                            ▼
┌─────────────────────────────────────────────────────────────┐
│  Tool（原子能力）                                             │
│  internal/tools/registry.go — 82 个已注册                    │
│       │                                                      │
│       ├── bespoke（Agent/Go 本地实现）                        │
│       └── HTTP catalog → GeeGooBot / Signal / Data           │
└───────────────────────────┬─────────────────────────────────┘
                            │
                            ▼
              GeeGoo 栈 HTTP（禁止转发旧 Trading Python）
```

| 维度 | Skill | Tool |
|------|-------|------|
| 粒度 | 多步工作流（盘前、盘后…） | 单次 API 调用或本地操作 |
| 触发 | `geegoo run`、scheduler | LLM tool_call 或 workflow 硬编码步骤 |
| 配置 | `skills/<name>/manifest.yaml` | `catalog/catalog.go` + `bespoke.go` |
| Chat 默认 | 不直接暴露（workflow 专用） | 按 toolset 白名单暴露给 LLM |

---

## Tools 系统

### 注册与数量

| 类型 | 数量 | 注册位置 |
|------|------|----------|
| HTTP 转发 | 62 | `tools/bootstrap.go` ← `catalog.AllHTTP()` |
| Bespoke 手写 | 21 | `tools/bespoke.go` |
| **合计（去重）** | **82** | `search_code` 由 bespoke 覆盖 HTTP 定义 |

**完整树形图（含实现状态）** → [../reference/geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md)

### Toolset（Hermes 风格分组）

定义：`internal/tools/toolset.go` + `domains.go`

| Toolset ID | 中文 | 默认 chat | 工具数 |
|------------|------|-----------|--------|
| `market` | 行情与分析 | ✅ | 17 |
| `strategy` | 策略生成与回测 | ✅ | 3 |
| `bot_manager` | 交易 Bot | ✅ | 21* |
| `reminder_manager` | 提醒 Bot | ✅ | 15 |
| `report_query` | 报告查询 | ✅ | 10 |
| `report_workflow` | 报告 Workflow | ❌ | 8 |

\*含规划中的 `switch_bot`（**未注册**）

切换：`/toolsets market,strategy` 或配置文件。

### 五类 Tool（设计taxonomy）

| 类 | 代表 Tool | 说明 |
|----|-----------|------|
| **Perception** | `search_code`, `get_current_price`, `fetch_*_news` | 感知市场与标的 |
| **Analysis** | `get_mcp_analysis`, `get_capital_flow`, `get_index_signals` | 技术与资金分析 |
| **Decision** | `recall`, `read_working_state`, `recall_yesterday_summary` | 辅助决策的记忆 |
| **Action** | `create_*_report`, `create_*_bot`, `save_local_report` | 写入与变更 |
| **Meta** | `write_execution_log`, `web_search` | 审计与兜底 |

**刻意不提供 Bash Tool** — 所有副作用走白名单。

### 关键机制

| 机制 | 文件 | 说明 |
|------|------|------|
| ApprovalGate | `approval.go` | chat 下 `create_/update_/delete_/switch_` 需用户确认 |
| ClassifyHTTPPayload | `contract.go` | API code=100 但 data 空 → `skipped` |
| NeedsMCPToken | `catalog/token.go` | 除 search/signals 外默认注入 mcp_token |
| HTTPBackends | `httpbackend.go` | 按 tool 名路由到 MCP :3120 / Signal :3200/3210 |

### HTTP 后端映射

| 客户端 | 端口 | Tools 示例 |
|--------|------|------------|
| GeeGooBot mcp-api | 3120 | 报告、Bot、资金、策略转发 |
| Signal signal-api | 3200 | `loopback_strategy`, `search_code` |
| Signal catalog-api | 3210 | `get_index_signals`, `get_signal_combinations` |
| GeeGooData | 3300 | 经 MCP 间接（现价、资金等） |

MCP 路由 SSOT：[../reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md)

### 对话时常用不了的能力

| Tool | 原因 |
|------|------|
| `fetch_market_news` / `fetch_stock_news` | Script runner 未接 → `skipped` |
| `get_ticker` / `get_broker` / `get_position` | 富途 Noop |
| `switch_bot` | 未注册 |
| `generate_dca_strategy` | 需 analyze-api 部署；参数要齐 |

---

## Skills 系统

### 内置 Skill 注册表

代码：`internal/skills/registry.go` + `loader.go`

| Skill | 状态 | 入口 | 说明 |
|-------|------|------|------|
| `pre_market` | ✅ 完整 | `geegoo run pre_market` | Phase A（指数+新闻）+ Phase B（逐股） |
| `intraday` | 📋 占位 | — | manifest 存在，步骤为空 |
| `post_market` | 📋 占位 | — | 同上 |

列出：`geegoo skills list`

### Skill 资源目录

```text
skills/
├── pre_market/
│   ├── SKILL.md              # 人类可读说明（Cursor Skill 同源）
│   ├── manifest.yaml         # tools 白名单、workflow 结构、rules 引用
│   ├── workflow.md           # 业务步骤指南
│   ├── template.md           # 报告 Markdown 模板
│   └── supervisor_checks.yaml
└── bundled/                  # 新闻等捆绑 Skill（供未来 script runner）
    ├── finance-news/
    └── eastmoney-news/
```

`manifest.yaml` 是**文档与审计 SSOT**；Go 侧步骤函数在 `internal/workflow/premarket.go` 注册到 `skills.Spec`。

### pre_market 步骤概要

**Phase A（全局一次）**

1. `check_trading_day` — 非交易日短路
2. `get_report_bot_codes` — 待分析标的列表
3. 五大指数 `get_mcp_analysis`（hourly）
4. `fetch_market_news` US/CN/HK

**Phase B（每股）**

1. `fetch_stock_news` → `get_capital_flow` + `get_capital_distribution`
2. `get_mcp_analysis`（weekly）→ `get_bot_yesterday_attitude`
3. `list_today_reports` 幂等检查
4. `report.Synthesizer`（LLM evidence-only）→ `create_pre_market_report` → `save_local_report`

每步：`Working.Apply` + `write_execution_log` + checkpoint。

### Skill vs Chat Toolset

| 场景 | 加载方式 |
|------|----------|
| `geegoo chat` | 默认 5 toolset；LLM 自选 Tool |
| `geegoo run pre_market` | workflow 硬编码步骤；**不**走 LLM 编排顺序 |
| Scheduler | 同 `run`；失败按 verdict 重试 |

盘前 MVP 选择**确定性 Workflow** 而非纯 ReAct，是为可恢复、可验收、可对标 Hermes cron 稳定性。

### 外部 Cursor Skills 映射

| 外部 Skill | Agent 内对应 |
|------------|--------------|
| `geegoo` 盘前/盘后 workflow | `skills/pre_market` + domains 映射 |
| `geegoo` 按需分析 | chat + `market` toolset |
| `finance-news` / `eastmoney-news` | bundled；待 script runner |
| `monday` 等 | 未内置；通过 MCP Tool 间接 |

详见 [domains/geegoo-agent-skill-mapping.md](./domains/geegoo-agent-skill-mapping.md)。

---

## 扩展指南

### 新增 Tool

1. HTTP 转发：在 `catalog/catalog.go` 的 `AllHTTP()` 增加 `HTTPSpec`
2. 或 bespoke：在 `bespoke.go` 中 `r.Register`
3. 更新 `domains.go` / `toolset.go` 分组
4. 同步 [geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md)
5. GeeGooBot 侧注册路由（若为新 MCP 端点）

### 新增 Skill

1. 创建 `skills/<name>/` 目录（SKILL.md + manifest.yaml）
2. 在 `internal/workflow/` 实现 `PhaseASteps` / `PerStockSteps`
3. 在 `skills/loader.go` 的 `RegisterBuiltins` 注册 `Spec`
4. 可选：在 `scheduler/jobs.go` 增加 cron 条目

### 规划未实现

| 名称 | 类型 | Phase |
|------|------|-------|
| `wait_for_human` | Tool | Bot 创建前确认 |
| `spawn_subagent` | Tool | 子 Agent |
| `fetch_global_quote` | Tool | 免费行情 |
| `on_demand_analysis` | Skill pack | chat 路由 |
| `bot_manager` | Skill pack | 与 toolset 合并中 |

---

## 延伸阅读

- [layers/L2-tools/registry.md](./layers/L2-tools/registry.md) — Registry API
- [layers/L2-tools/tool-catalog.md](./layers/L2-tools/tool-catalog.md) — 全量 Tool 清单（设计态 ~87）
- [layers/L5-application/skills.md](./layers/L5-application/skills.md) — SkillLoader 细节
- [domains/skills-and-tools-taxonomy.md](./domains/skills-and-tools-taxonomy.md) — 分类学
