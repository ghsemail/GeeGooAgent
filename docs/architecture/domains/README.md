# Domains — GeeGoo 领域映射

连接 **外部 GeeGoo 生态**（MCP API、Cursor Skills）与 **Agent 内部**（L2 Tool 名、L5 Workflow 步骤）。

本目录不是运行时 Go 包，而是**领域知识 SSOT**。

## 在架构中的位置

```text
外部 Cursor Skills（geegoo、finance-news…）
        ↓ 提炼
domains/*.md（本目录）
        ↓ 实现
internal/tools + internal/workflow + skills/
        ↓ 调用
GeeGooBot :3120 / GeeGooSignal :3200-3230 / GeeGooData :3300
```

> **架构原则**：GeeGoo 栈纯 Go 原生，**禁止** HTTP 转发旧 Trading Python（5600/5700）。

## 文档索引

| 文档 | 回答的问题 |
|------|------------|
| [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md) | Skill vs Tool 怎么分 |
| [geegoo-api-routing.md](./geegoo-api-routing.md) | 多端口路由、报告查询优先级 |
| [geegoo-agent-skill-mapping.md](./geegoo-agent-skill-mapping.md) | 盘前 workflow 步骤 → Tool 顺序 |
| [geegoo-skill-mapping.md](./geegoo-skill-mapping.md) | geegoo Skill 章节 → Tool 名 |
| [tradingbot-tools-index.md](./tradingbot-tools-index.md) | Tool 分类索引 |

## 外部 SSOT

| 资源 | 路径 |
|------|------|
| MCP HTTP 73 路由 | [../reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md) |
| Agent Tool 可用性树 | [../reference/geegoo-agent-tools-tree.md](../reference/geegoo-agent-tools-tree.md) |
| GeeGooBot 已实现路由 | GeeGooBot `docs/api/implemented-routes.md` |

## 设计原则

| 原则 | 说明 |
|------|------|
| 单一事实来源 | API 路由以 interface-map 为准；domains 解释「怎么用」 |
| Tool 名稳定 | HTTP camelCase → snake_case，减少 LLM 幻觉 |
| Skill 是规范上游 | Cursor Skill 更新时先改 domains，再同步 L2/L5 |

## 与 Tools & Skills 文档

综合导读 → [../tools-and-skills.md](../tools-and-skills.md)

## 服务端口速查

| 服务 | 端口 | Agent 用途 |
|------|------|------------|
| GeeGooBot mcp-api | 3120 | 报告、Bot、资金、策略转发 |
| GeeGooSignal signal-api | 3200 | search_code、loopback |
| GeeGooSignal catalog-api | 3210 | 指标/组合信号 |
| GeeGooSignal analyze-api | 3230 | getMCPAnalysis、generate_* |
| GeeGooData data-api | 3300 | 现价、资金（经 MCP 间接） |
| GeeGooAgent runtime | 3400 | HTTP chat 入口 |
