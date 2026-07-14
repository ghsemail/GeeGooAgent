# L2 — Tool & MCP Layer

Agent 与 GeeGoo 生态的**唯一接口**。Runtime 不手写 HTTP；所有外部 IO 经 Tool Registry。

> **快速查阅**：[tools-and-skills.md](../../tools-and-skills.md) · [geegoo-agent-tools-tree.md](../../../reference/geegoo-agent-tools-tree.md)

## 实现概览（Go）

| 模块 | 文件 | 说明 |
|------|------|------|
| Registry | `internal/tools/registry.go` | 注册、Schema、Execute |
| HTTP 转发 | `internal/tools/bootstrap.go` + `catalog/` | 62 个 MCP 转发 Tool |
| Bespoke | `internal/tools/bespoke.go` | 21 个手写 Tool |
| Toolset | `internal/tools/toolset.go` | Hermes 风格分组 |
| 审批门控 | `internal/tools/approval.go` | 危险写操作 |
| 契约分类 | `internal/tools/contract.go` | 空成功检测 |
| MCP Client | `internal/clients/mcp/` | Bearer + mcp_token |

**已注册：82**（2026-07）

## 核心设计决策

| 决策 | 理由 |
|------|------|
| 中央 Registry | Skill / toolset 按名过滤；Scheduled 不暴露 Bot 写 |
| 五类分层 | Perception → Analysis → Decision → Action → Meta |
| GeeGoo 3xxx 客户端 | 统一出站；**禁止**转发旧 Trading Python |
| `Result` 信封 | `status/summary/data/meta`；Executor 写 Working + Evidence |
| 无 Bash Tool | 股票 Agent 不需要任意 shell |

## 数据流

```text
ReActLoop / workflow.Runner
    └── Registry.Execute(name, args, toolCtx)
            ├── bespoke.Handle → 本地 / 直连 Signal
            └── HTTP catalog → HTTPBackends.ForTool(name)
                    ├── MCP :3120（默认）
                    ├── Signal :3200（search_code, loopback）
                    └── Catalog :3210（signals）
```

## 模块索引

| 文档 | 内容 |
|------|------|
| [registry.md](./registry.md) | Registry API、过滤、执行 |
| [tool-catalog.md](./tool-catalog.md) | 设计态全量 ~87 Tool |
| [clients.md](./clients.md) | MCP 客户端、鉴权、端点 |
| [tool-server-mapping.md](./tool-server-mapping.md) | Tool → 服务端口 |
| [sandbox-integration.md](./sandbox-integration.md) | WorkspaceGuard、路径边界 |

## Toolset 与 Chat

默认 chat 加载 `market` + `strategy` + `bot_manager` + `reminder_manager` + `report_query`。

`report_workflow` 仅 `/toolsets report_workflow` 或跑 `geegoo run` 时启用。

## 实现状态摘要

| 状态 | 数量 | 示例 |
|------|------|------|
| ✅ 可用 | ~68 | search、现价、Bot CRUD、报告 CRUD |
| ⚠️ 部分 | ~14 | 新闻 skipped、富途三接口 Noop、简化策略/分析 |
| ❌ 未注册 | 11 | switch_bot、wait_for_human、fetch_global_quote |

详见 [tools-tree](../../../reference/geegoo-agent-tools-tree.md)。

## 边界

- **提供**：Tool 注册、执行、Schema、HTTP 客户端
- **不提供**：Workflow 顺序（L4）、报告模板（L5 `skills/`）
- **不提供**：LLM Prompt 组装（L4/L5）

## 扩展

见 [tools-and-skills.md §扩展指南](../../tools-and-skills.md#扩展指南)。
