# GeeGooAgent 文档

## 架构（SSOT）

**[architecture/](./architecture/)** — 完整架构，从 [README.md](./architecture/README.md) 进入。

| 优先阅读 | 文件 |
|----------|------|
| 系统总览 | [architecture/overview.md](./architecture/overview.md) |
| **Agent Runtime 定稿** | [architecture/agent-runtime-architecture.md](./architecture/agent-runtime-architecture.md) |
| **Runtime 改造计划** | [architecture/agent-runtime-migration-plan.md](./architecture/agent-runtime-migration-plan.md) |
| **实现 / 未实现** | [architecture/implementation-status.md](./architecture/implementation-status.md) |
| Tool 运行态 | [architecture/layers/L2-tools/tools-status.md](./architecture/layers/L2-tools/tools-status.md) |

六层文档均在 `architecture/layers/L0-infrastructure/` … `L5-application/`。

## 其他

| 目录 | 用途 |
|------|------|
| [benchmark/](./benchmark/) | GeeGooAgent vs Hermes vs Grok Build 对标；**Agent Loop 专项**见 [benchmark/agent-loop/](./benchmark/agent-loop/) |
| [engineering/](./engineering/) | 编码规范、需求、Cursor 工作流 |
| [reference/geegoo-mcp/](./reference/geegoo-mcp/) | MCP HTTP 参考 |
| [archive/](./archive/) | 开发计划稿、旧重定向（非架构正文） |
