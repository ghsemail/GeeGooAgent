# L5 — Application Layer

应用层定义「**跑什么任务**」：Skill 包、CLI 入口、触发方式、Rules/Prompts。

> Go 实现：`cmd/geegoo`、`internal/skills`、`skills/`、`rules/`、`internal/chatprompt`

## 模块索引

| 模块 | 文档 | Go 代码 | 状态 |
|------|------|---------|------|
| **Skills** | [skills.md](./skills.md) | `internal/skills`, `skills/` | pre_market ✅ |
| **Tools**（L2） | [../L2-tools/README.md](../L2-tools/README.md) | `internal/tools` | 82 注册 |
| 入口点 | [entrypoints.md](../../entrypoints.md) | `cmd/geegoo` | ✅ |
| 触发 | [triggers.md](./triggers.md) | scheduler + CLI | ✅ |
| Rules & Prompts | [rules-prompts.md](./rules-prompts.md) | `chatprompt`, `rules/` | ✅ |
| Subagent | [subagents.md](./subagents.md) | — | ❌ 未实现 |

## 运行模式

| 模式 | 入口 | 编排 | Tool 集 |
|------|------|------|---------|
| **Interactive** | `geegoo chat` | ReAct | 默认 5 toolset |
| **Scheduled** | `geegoo scheduler` → `run pre_market` | Workflow | manifest 白名单 |
| **HTTP** | `:3400` chat/completions | ReAct | 配置 toolset |
| Signal | webhook | ❌ 未实现 | — |

## Skill Pack 结构

```text
skills/pre_market/
├── SKILL.md
├── manifest.yaml
├── workflow.md
├── template.md
└── supervisor_checks.yaml
```

## 双能力来源

| 来源 | 用途 | 实现 |
|------|------|------|
| **Skill**（工作流） | 盘前/盘后定时任务 | `geegoo run` + workflow |
| **Toolset**（对话） | 按需分析、策略、Bot | `geegoo chat` + LLM |

外部 Cursor Skills（`geegoo`、`finance-news`）映射见 [domains/](../../domains/)。

## 边界

- **提供**：任务定义、CLI、Rules 文本、报告模板路径
- **不提供**：ReAct 循环（L4）、HTTP 客户端（L2）、SQLite（L0）

## 延伸阅读

- [../../overview.md](../../overview.md)
- [skills.md](./skills.md)
