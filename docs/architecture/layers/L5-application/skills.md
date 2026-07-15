# L5 — Skill 系统

> Go 实现：`internal/skills/` + `skills/` 资源目录

## 职责

- 定义**任务边界**（盘前 / 盘后 / 盘中）
- 提供 workflow 步骤、报告模板、supervisor 检查项
- 供 `geegoo run <skill>` 与 Scheduler 调度

Skill **不是** LLM 可调用的 function；它是运维与开发侧的**任务包**。

## 架构关系

```text
skills/pre_market/manifest.yaml     ← 人类可读 SSOT（tools 白名单、workflow 结构）
        ↓ 对齐
internal/skills/loader.go           ← RegisterBuiltins → Spec
        ↓ 引用
internal/workflow/premarket.go      ← PhaseASteps / PerStockSteps
        ↓ 执行
workflow.Runner.Run(skill)
```

Tool 体系（toolset、注册、可用性）→ [L2-tools/README.md](../L2-tools/README.md)

## Spec 结构（Go）

```go
// internal/skills/registry.go

type Spec struct {
    Name         string
    Description  string
    PhaseA       func() []workflow.Step
    PerStock     func() []workflow.Step
    TemplatePath string   // skills/pre_market/template.md
    ManifestPath string   // skills/pre_market/manifest.yaml
}
```

## 内置 Skill

| Name | PhaseA | PerStock | 状态 |
|------|--------|----------|------|
| `pre_market` | `workflow.PhaseASteps` | `workflow.PerStockSteps` | ✅ |
| `intraday` | `emptySteps` | `emptySteps` | 📋 占位 |
| `post_market` | `emptySteps` | `emptySteps` | 📋 占位 |

列出：`geegoo skills list`

## 资源目录结构

```text
skills/<name>/
├── SKILL.md                 # 描述、触发条件（Cursor Skill 同源）
├── manifest.yaml            # tools[]、workflow 结构、rules、bundled
├── workflow.md              # 步骤业务说明
├── template.md              # 报告 Markdown 模板
└── supervisor_checks.yaml   # 机器可读验收项
```

### manifest.yaml 示例（pre_market）

- `tools[]` — 文档白名单（~19 MVP 工具）
- `workflow.prelude / phase_a / phase_b` — 与 Go 步骤对齐
- `rules[]` — 指向 `rules/` 下常驻规则
- `bundled[]` — 新闻 Skill 路径（待 script runner）

## 与 Chat Toolset 的区别

| | Skill | Toolset |
|---|-------|---------|
| 用于 | `geegoo run`、scheduler | `geegoo chat` |
| 编排 | 确定性步骤（Go） | LLM ReAct 自选 Tool |
| 配置 | `skills/*/manifest.yaml` | `toolset.go` |

`on_demand_analysis`、`strategy`、`bot_manager` 在蓝图中是 Skill Pack 名；当前实现中其能力主要由 **chat toolset** 提供。

## System Prompt 组装

Chat 路径（非 workflow）：

```text
chatprompt.Build()           # 稳定 system：人格 + Tool 路由 + 记忆规则
+ rules/（逻辑引用，非每轮重载）
+ 用户消息
+ RuntimeMessages() 动态 context（user 角色注入）
```

Workflow 路径不经过 LLM 选步；仅在 `report.Synthesizer` 阶段调用 LLM。

## Supervisor

`skills/pre_market/supervisor_checks.yaml` 定义检查项；运行时由 `workflow/supervisor.go` 执行：

- phase 完成标记
- 本地 md 存在
- API report 字段
- evidence_refs 非空

Verdict 驱动 scheduler 退避重试。

## 扩展新 Skill

1. 创建 `skills/<name>/`（SKILL.md + manifest.yaml + template）
2. 在 `internal/workflow/` 实现步骤函数
3. `loader.go` → `RegisterBuiltins` 增加 `Spec`
4. 可选：`scheduler/jobs.go` 增加 cron

## 外部 Skill 映射

| Cursor / geegoo Skill | Agent 内 |
|-----------------------|----------|
| geegoo 盘前 workflow | `pre_market` |
| finance-news | `skills/bundled/finance-news` |
| geegoo 按需分析 | chat + `market` toolset |

→ [domains/geegoo-skill-mapping.md](../../domains/geegoo-skill-mapping.md)

## 延伸阅读

- [triggers.md](./triggers.md) — 触发模式
- [rules-prompts.md](./rules-prompts.md) — Rules 与 Prompt
- [subagents.md](./subagents.md) — 子 Agent（规划）
