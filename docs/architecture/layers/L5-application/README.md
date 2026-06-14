# L5 — Application Layer

应用层定义「跑什么任务」：Skill 包、触发方式、Prompt/Rules、Subagent 委派。

## 模块设计说明

L5 是用户与运维**直接感知**的一层：定时盘前、cron 替代、聊天按需分析、Bot 创建确认——都通过本层的 Skill Pack 与触发模式表达。L4 Runtime 是通用引擎；L5 注入领域知识（GeeGoo workflow、报告模板、API 路由规则）。

**核心设计决策**

| 决策 | 理由 |
|------|------|
| Skill = 目录包（SKILL.md + workflow + tools 清单） | 对齐 Cursor/Hermes skill 习惯；便于从 `geegoo` skill 迁移 |
| Rules 与 Skill 分离 | Rules 常驻（禁止硬编码代码、attitude 映射）；Skill 按任务切换 |
| 三触发模式 | Scheduled 无 Bot 写权限；Interactive 全量 Tool + `wait_for_human` |
| 双 Skill 来源合一 | `geegoo`→盘前/盘后 workflow；`geegoo`→分析/Bot/策略，由 L2 路由到正确端口 |
| Subagent 受限委派 | StockAnalyst 并行个股分析；禁止嵌套 spawn，max_steps 更低 |

**Skill Pack 结构（约定）**

```text
skills/pre_market/
├── SKILL.md           # 目标、工具子集、约束
├── workflow.md        # 阶段 A/B 业务说明（给 Planner 的软指南）
└── manifest.yaml      # tools[], rules[], max_steps（可选）
```

**触发 → Runtime 映射**

```text
systemd timer ──▶ CLI run pre_market ──▶ mode=scheduled ──▶ tools 子集
webhook       ──▶ CLI run intraday   ──▶ mode=signal
geegoo-agent chat ──▶ 意图路由 skill    ──▶ mode=interactive
```

**边界**

- **提供**：业务任务定义、Prompt/Rules 文本、入口 CLI、Subagent 规格
- **不提供**：循环实现、API 客户端、记忆存储（归 L4/L2/L3）
- **迁移来源**：`~/.cursor/skills/geegoo` 的 workflow、template、api-routing

**MVP 范围**

仅 `skills/pre_market/` + `CLI run pre_market` + Scheduled 模式 + 全套 rules（`bot-creation` 可 stub）。

## 模块索引

| 模块 | 文档 | 代码 |
|------|------|------|
| Skill 系统 | [skills.md](./skills.md) | `runtime/skill_loader.py`, `skills/` |
| 触发入口 | [triggers.md](./triggers.md) | `cli.py`, `runtime/triggers.py` |
| Rules & Prompts | [rules-prompts.md](./rules-prompts.md) | `prompts/`, `rules/` |
| Subagent | [subagents.md](./subagents.md) | `subagents/` |

## 运行模式

| 模式 | 入口 | 加载 Skill | Tool 集 |
|------|------|------------|---------|
| Scheduled | systemd timer | pre_market / post_market | 感知+分析+报告 |
| Signal | webhook | intraday | +持仓+信号 |
| Interactive | `geegoo-agent chat` | 意图路由 | 全量（Bot 需确认） |

## MVP

仅实现 `skills/pre_market/` + CLI `run pre_market` + Scheduled 模式。
