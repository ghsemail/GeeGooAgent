# L2 — Tool & MCP Layer

Agent 与外部世界的唯一接口（GeeGoo API = MCP 等价物）。

## 模块设计说明

L2 把 GeeGoo 生态的 HTTP API、新闻脚本、本地文件操作封装为 **LLM 可调用的 Typed Tool**。对 Runtime 而言，GeeGoo MCP 服务等价于 Claude Code 的 Read/Bash/Grep——但 GeeGoo Agent **刻意不提供 Bash**，所有副作用走白名单 Tool。

**核心设计决策**

| 决策                            | 理由                                                              |
| ----------------------------- | --------------------------------------------------------------- |
| ToolRegistry 集中注册             | Skill Pack 按模式加载子集；Scheduled 模式不暴露 Bot create/delete            |
| 五类分层（Perception→Meta）         | 约束 Planner 先感知再分析再写入，减少跳步                                       |
| GeeGoo 3xxx HTTP 客户端 | 统一 GeeGooBot mcp-api + 信号服务；Tools 不拼 URL                                 |
| ToolResult 信封                 | 统一 `status/summary/data`，Executor 写回 WorkingMemory 与 Checkpoint |
| Schema 硬校验                    | 如 `create_pre_market_report` 缺 `confidence` 则在 L2 拒绝，不浪费 API    |

**数据流**

```text
Executor (L4)
    └── ToolRegistry.get(name)
            └── Tool 实现
                    ├── GeeGooBot mcp-api (:3120)   workflow / 资金 / 报告
                    ├── GeeGooBot mcp-api (:3120)  分析 / Bot / 策略
                    ├── GeeGooSignal catalog-api (:3210)   指标信号
                    ├── 本地脚本               新闻 / 行情
                    └── SandboxManager         路径与网络校验
```

**边界**

- **提供**：~87 个 Tool（MVP 19）、HTTP Clients、参数校验、结果格式化
- **不提供**：Workflow 顺序（L4/L5）、报告 Markdown 渲染（L5 模板 + LLM）、记忆语义（L3）
- **与 Skill 关系**：`pre_market` 只注册 catalog 中对应子集；`bot_manager` 才加载全量 CRUD

**MVP 范围**

盘前 Tool 子集 + `MarketClient`/`GeeGooBotClient` + `fetch_*_news` + `create_pre_market_report` + `get_capital_flow`/`get_capital_distribution`。

## 模块索引

| 模块           | 文档                                                 |
| ------------ | -------------------------------------------------- |
| ToolRegistry | [registry.md](./registry.md)                       |
| 工具目录         | [tool-catalog.md](./tool-catalog.md)               |
| HTTP Clients | [clients.md](./clients.md)                         |
| Sandbox 集成   | [sandbox-integration.md](./sandbox-integration.md) |

## 五类 Tool

Perception → Analysis → Decision → Action → Meta

## 设计约束

- **无 Bash Tool**
- 所有 IO 经 Registry
- Clients 不对 Runtime 暴露

## MVP

盘前 Tool 子集 + 双端口 Clients。