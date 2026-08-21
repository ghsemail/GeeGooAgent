# L5 — Rules & Prompts

## 职责

常驻指令层：身份、硬规则、报告格式——不随 Skill 变化的部分。

## 三层指令（Chat）

GeeGoo 将「谁在说」「偏好什么」「什么绝对不能违反」分开，避免与 Codex 单文件 `AGENTS.md` 或 Workflow `StockWorkspace` 混用。

| 层 | 来源 | 维度 | 可编辑 |
|----|------|------|--------|
| **Soul** | `SOUL.md` / `defaultSoulText` | 租户（`tenants/{userId}/SOUL.md`） | ✅ Dashboard / 手改 |
| **Context Profile** | `AGENTS.md` | 租户 + 可选 stock / bot / strategy | ✅ 见下表 |
| **Hard rules** | 代码 embed + `rules/*.md` SSOT | 全局 | ❌ 仅发版 |
| **Tool routing** | `chatprompt` Go 字符串 | 全局 | ❌ 仅发版 |

Workflow Skill **不加载** Context Profile（确定性步骤 + Supervisor）；Workflow 内按股的运行态见 `memory.StockWorkspace`（非本页 AGENTS.md）。

### Context Profile 路径

| type | key | 文件路径 |
|------|-----|----------|
| `global` | `default` | `{GEEGOO_HOME}/AGENTS.md` |
| `user_default` | `{userId}` | `{GEEGOO_HOME}/tenants/{userId}/AGENTS.md` |
| `market` | `CN` / `HK` / `US` | `{GEEGOO_HOME}/tenants/{userId}/markets/{market}/AGENTS.md` |
| `stock` | `00700.HK` | `{GEEGOO_HOME}/tenants/{userId}/stocks/{code}/AGENTS.md` |
| `automation` | `{botId}` | `{GEEGOO_HOME}/tenants/{userId}/automations/{botId}/AGENTS.md` |

分析 Prompt 偏好写在 **stock** profile 内，不单独建 type。

**同一用户可有多个 Profile**；Chat 会话通过 `metadata.context_profiles`（如 `["market:HK","stock:00700.HK","automation:bot-id"]`）激活。

**P1-4 MVP** 已实现 global + user_default + 会话多 profile；路径与 API 见 runtime `GET/PUT /v1/context/profiles`。

### Chat system 组装顺序

```text
Soul → global AGENTS → user AGENTS → session profiles → hard rules → ToolRouting → Memory → Endpoints
```

`SyncChatSystemPrompt` 仅在 profile 文件变更或会话切换 profile 时更新 system 内容，以保持 prefix cache 友好。

## 仓库内 rules/（Skill / verify SSOT）

文档与质检用；**不**在 Chat 运行时整包读入 system（硬规则由代码选择性 embed）。

```
prompts/
└── identity.md              # 设计参考（实现见 soul.go / SOUL.md）

rules/
├── api-routing.md           # 3120；资金流向/报告查询路由
├── attitude-mapping.md      # bullish→long 等
├── report-format.md         # 九章盘前模板约束
├── execution-log.md         # 日志规范
├── risk-disclaimer.md       # 免责声明
├── analysis.md              # getMCPAnalysis period/name 约束
├── bot-creation.md          # Phase 6：创建前确认
└── signal-reference.md      # Phase 5：指标信号
```

## identity / Soul 要点

- 股票分析专员身份
- 禁止编造行情
- 禁止硬编码股票代码
- 港股新闻可无数据

## Context Profile（AGENTS.md）示例

`tenants/ghsemail/stocks/00700.HK/AGENTS.md`:

```markdown
- 默认结合资金分布 + 周线 MCP 分析
- 摘要控制在 300 字内
- 不要主动推荐创建 GRID Bot
```

不得写入「可跳过 confidence」「可编造价格」等与 Soul / hard rules 冲突的内容。

## Rules 与 Tool Schema 分工

| 层级          | 约束方式       | 示例                     |
| ----------- | ---------- | ---------------------- |
| Context Profile | Prompt 软约束 | 「本 Bot 只讨论定投，别提网格」 |
| Hard rules  | Prompt + Supervisor | report 九章结构、result 枚举 |
| Tool Schema | 硬拒绝        | 缺 `confidence` 不发给 API |
| Supervisor  | 跑后检查       | 每股都有 report_id         |

## 与其它「workspace」名词

| 名词 | 包/配置 | 用途 |
|------|---------|------|
| `WorkspaceRoot` | `output_dir` | 报告文件、execution_log 落盘 |
| `StockWorkspace` | `memory.StockWorkspace` | Skill phase B 运行态 |
| **Context Profile** | `AGENTS.md` | Chat 可编辑规则（本页） |

## 实现状态

| 项 | 状态 |
|----|------|
| Soul / SOUL.md | ✅ |
| ToolRouting 硬编码 | ✅ |
| Context Profile / AGENTS.md | ✅ P1-4 |
| 多 profile 按会话 | ✅ P1-4 |
| rules/*.md → Chat 动态加载 | ❌ 不做（仅 embed 硬规则子集） |

详细排期：[codex-inspired-roadmap.md](../../../benchmark/agent-loop/codex-inspired-roadmap.md) § P1-4。
