# L4 — Workflow Engine

确定性工作流编排：**非** LLM 选步，按 Skill 注册的 Phase A / PerStock 步骤顺序执行。

> Go 实现：`internal/workflow/runner.go`、`premarket.go`、`supervisor.go`

## 职责

| 能力 | 说明 |
|------|------|
| 步骤执行 | 每步 `Registry.Execute` + `Working.Apply` |
| 幂等 resume | 按 `CompletedStepKeys` 跳过已完成步骤 |
| Checkpoint | 每步落盘，支持 `geegoo resume` |
| Supervisor | 跑后 verdict → scheduler 退避 |
| 报告合成 | `report.Synthesizer` → `create_pre_market_report` |

## 入口

```text
geegoo run pre_market
  → internal/app.App.RunSkill(name)
  → skills.Registry.Get(name)
  → workflow.Runner.Run(spec)
```

## Runner 核心流程

```text
Run(skill Spec):
  PhaseA = skill.PhaseA()
  for step in PhaseA:
      processStep(step)   // 可 skip 若 key 已在 CompletedStepKeys

  bots = Working.ReportBots
  for bot in bots:
      steps = skill.PerStock()
      for step in steps:
          processStep(step)

  finishWithSupervisor()
  synthesizeAndSubmitReports()
```

### processStep

1. 若 `DryRun` → 跳过写操作
2. `tools.Execute(step.Tool, args)`
3. `memory.Working.Apply(result)`
4. `write_execution_log`
5. `checkpoint.Save` + 记录 `CompletedStepKeys`

### 错误分类

`workflow/errors.go`：

| 类型 | 行为 |
|------|------|
| Recoverable | 自动重试 1 次 |
| Terminal | 立即 fail run |

## 与 ReAct 对比

| | Workflow | ReAct |
|---|----------|-------|
| 编排 | Go 硬编码 | LLM tool_calls |
| 用于 | pre_market | geegoo chat |
| 可预测性 | 高 | 灵活 |
| resume | step key 幂等 | 会话历史 |

## pre_market 步骤

定义：`internal/workflow/premarket.go`  
文档对齐：`skills/pre_market/manifest.yaml`  
领域映射：[domains/geegoo-skill-mapping.md](../../domains/geegoo-skill-mapping.md)

## Supervisor

`finishWithSupervisor` → `Supervisor.Verify`：

| Verdict | 含义 |
|---------|------|
| `pass` | 完成 |
| `recoverable` | 缺步可补跑 |
| `terminal` | 失败停手 |

非交易日直接 `pass`。

## 报告合成

`internal/report/synthesis.go`：

- LLM 只写 reason / suggestion / summary（evidence-only）
- `result` / `confidence` 规则锁定
- LLM 失败 → 规则回退，不阻塞 workflow

## 测试

- `premarket_test.go`
- `resume_test.go`
- `supervisor_test.go`
- `synthesis_fallback_test.go`

## 延伸阅读

- [agent-loop.md](./agent-loop.md) — ReAct 路径
- [../L2-tools/README.md](../L2-tools/README.md) — Tool 与 Skill 分工
- [../../cross-cutting/supervisor.md](../../cross-cutting/supervisor.md)
