# Domains — GeeGoo 领域映射

连接 Cursor Skill、GeeGoo MCP API 与 Agent 内部 Tool / Workflow。

| 文档 | 内容 |
|------|------|
| [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md) | Skill vs Tool vs Bundled 分类 |
| [geegoo-skill-mapping.md](./geegoo-skill-mapping.md) | geegoo SKILL → Tool、资产对齐、交互规范 |
| [geegoo-api-routing.md](./geegoo-api-routing.md) | 3120/3200/3210/3230 路由 |
| [geegoodata-news.md](./geegoodata-news.md) | **新闻聚合**（目标：GeeGooData 多源 + `news_sources.xml`） |

| 外部 SSOT | 路径 |
|-----------|------|
| MCP HTTP 73 路由 | [reference/geegoo-mcp/interface-map.md](../../reference/geegoo-mcp/interface-map.md) |
| Tool 运行态 | [layers/L2-tools/tools-status.md](../layers/L2-tools/tools-status.md) |

```text
geegoo Skill  →  domains/*.md  →  internal/tools + workflow  →  GeeGoo :3120+
```

> 禁止 HTTP 转发旧 Trading Python（5600/5700）。
