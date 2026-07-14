# L2 — Sandbox 集成

所有 Tool 执行**必须**经 L0 `SandboxManager`——Executor 不得直接 `tool.run()`。

## 调用链

```text
Executor.execute(call)
  → SandboxManager.execute(tool, call, ctx)
      → PolicyGate → WorkspaceGuard → ResourceGuard
      → tool.run()
      → Envelope.wrap() → ToolResult
  → working.apply(result)
  → bus.emit("ToolCompleted", ...)
```

## Executor 与 Sandbox 分工

| 组件                 | 职责                               |
| ------------------ | -------------------------------- |
| **Executor**       | 解析 call、选 tool、更新 Memory、发 Event |
| **SandboxManager** | 隔离、权限、资源、统一 ToolResult           |

## ToolResult 与 Runtime 决策

| exit_code / status      | Runtime 行为          |
| ----------------------- | ------------------- |
| `ok`                    | 继续 Loop             |
| `error` 可重试             | LLM 见 stderr，决定是否重试 |
| `error` 404 attitude    | 工具内已转 neutral，继续    |
| `dry_run`               | 跳过 HTTP，记日志         |
| `sandbox_layer=network` | 告警，不泄露内网            |

## 详见

[L0-infrastructure/sandbox.md](../../L0-infrastructure/sandbox.md) — 六层模型与 SandboxManager 完整设计。

## MVP

`SandboxManager` + PolicyGate + ResourceGuard + Envelope；Network allowlist 在 `clients/base.py` 同步实现。