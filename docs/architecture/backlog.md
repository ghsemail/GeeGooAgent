# 架构待办

> **唯一待办清单**。已定稿架构见 [agent-runtime-architecture.md](./agent-runtime-architecture.md)；实现状态见 [implementation-status.md](./implementation-status.md)。

Agent Runtime 控制面改造（Cognition / Model Policy / Memory port / Advisor / 包边界）**已完成**。下列为后续单独立项，不阻塞当前生产路径。

---

## Agent OS / 平台

| 项 | 说明 | 依赖 |
|----|------|------|
| **Web / Flutter Dashboard** | 纯 `runtimeapi` 客户端：会话列表、任务、doctor、配置；无 agent 逻辑 | 立项时写 `dashboard-client.md`（仅 API 契约） |
| **向量库 / Semantic Memory 后端** | 挂在 `memport.Port` 下；Session SSOT 仍为 SQLite | 业务需求明确后选型（Chroma / sqlite-vss 等） |
| **IDE 扩展** | 同 runtimeapi 消费方；优先级可高于 Flutter | Dashboard 契约或独立 API 设计 |
| **Cost Manager** | 会话级 token / 费用账单 | Model Policy + session 元数据 |
| **多租户** | 隔离 config / session / 配额 | Cost + auth 设计 |
| **Webhook 触发** | HTTP 入口触发 skill / chat | runtimeapi 扩展 |
| **Notify Gateway** | 飞书等通知经 GeeGooBot `internal/notify`，非 Agent 直连 webhook | GeeGooBot 侧实现；Agent 薄转发 |

---

## 业务能力

| 项 | 状态 | 说明 |
|----|------|------|
| `fetch_*_news` script runner | ❌ | 无 Python script runner；Go RSS/东财回退已覆盖大部分场景 |
| Bot 侧 scheduler | ❌ | 创建 Bot 后不自动跑；GeeGooBot 架构缺口 |
| Prompt 模板高级能力 | ⚠️ | Tool 已注册；依赖 catalog-api 稳定性 |
| `get_mcp_analysis` 质量 | ⚠️ | 依赖 analyze-api 部署与模型 |

完整 Tool 运行态 → [layers/L2-tools/tools-status.md](./layers/L2-tools/tools-status.md)。

---

## 可选增强（低优先级）

| 项 | 说明 |
|----|------|
| `internal/` 物理目录改名 | 如 `agent` → `agentkernel`；逻辑边界已由 `archboundaries` 保证 |
| `ComplexityPolicy.ToolSchemaThreshold` | 默认 0（仅 `TaskComplex` 抬 token）；可按环境配置正阈值 |
| 分布式 Tracing | 当前为轻量日志；见 L0 tracing 规划 |

---

## 维护约定

- 新能力**落地后**：更新 [implementation-status.md](./implementation-status.md) 与相关层 README，并从本文件删除对应行。
- 新规划项**只加在本文件**，不再新建「改造计划 / Phase 路线图」类中间文档。
