# Domains — 领域映射

GeeGoo 生态相关的**端口路由、Skill 能力对照**——连接外部 MCP 世界与内部 L2 Tool 命名。

## 模块设计说明

`domains/` 不是运行时模块，而是**领域知识层**：把 `geegoo` / `geegoo` 两个 Cursor Skill 里的隐含约定（哪个接口走 geegoo mcp :5700、哪些字段必填）提炼为 Agent 实现可引用的规范，避免散落在 L2/L5 文档里重复且不一致。

**文档分工**

| 文档 | 回答的问题 |
|------|------------|
| [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md) | **Skill vs Tool 怎么分**（盘前/盘中/盘后 = Skill，其余 = Tool 池） |
| [tradingbot-tools-index.md](./tradingbot-tools-index.md) | Tool 分类（详见 SSOT interface-map） |
| [../reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md) | **TradingBot SSOT 镜像** |
| [geegoo-api-routing.md](./geegoo-api-routing.md) | 双端口路由表、报告查询优先级、历史 bug 修复后的正确用法 |
| [geegoo-skill-mapping.md](./geegoo-skill-mapping.md) | `geegoo` SKILL 章节 → L2 Tool 名、Phase 归属 |
| [geegoo-agent-skill-mapping.md](./geegoo-agent-skill-mapping.md) | `geegoo` workflow 步骤 → Tool 调用顺序与 MVP 对齐 |

**设计原则**

| 原则 | 说明 |
|------|------|
| 单一事实来源 | API 路由以本目录为准；`clients.md` / `tool-catalog.md` 引用而非复制矛盾版本 |
| Skill 是规范上游 | Cursor Skill 更新（如 2026-05-20 修复 getCapitalFlow）时先改 domains，再同步 L2 |
| Tool 名稳定 | HTTP 路径 camelCase → Tool snake_case，映射表固定，减少 Planner 幻觉 |

**数据流（概念）**

```text
geegoo Skill（外部）
        ↓ 提炼
domains/*.md（本目录）
        ↓ 实现
L2 clients.py + tools/*.py
        ↓ 加载
L5 skill manifest tools[]
```

**MVP 范围**

完整维护 `geegoo-api-routing` + `geegoo-agent-skill-mapping`（盘前路径）；`geegoo-skill-mapping` 全量对照供 Phase 6 Bot 实现时查阅。

## 文档索引

- [skills-and-tools-taxonomy.md](./skills-and-tools-taxonomy.md)
- [tradingbot-tools-index.md](./tradingbot-tools-index.md)
- [geegoo-api-routing.md](./geegoo-api-routing.md)
- [geegoo-skill-mapping.md](./geegoo-skill-mapping.md)
- [geegoo-agent-skill-mapping.md](./geegoo-agent-skill-mapping.md)
