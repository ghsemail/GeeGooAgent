# L2 — ToolRegistry

> Go 实现：`internal/tools/registry.go`、`spec.go`、`policy.go`、`bootstrap.go`

## 职责

- 注册全部 Tool（当前 **82**）
- 按 toolset / chat 白名单过滤 Schema
- 导出 JSON Schema 供 LLM function calling
- 执行 Tool 并返回统一 `Result`
- **ToolSpec**：权限、并发、超时、结果上限在 `Register` 时解析

## 核心类型

```go
type ToolSpec struct {
    Policy          PolicyAction   // allow | prompt | forbidden
    ReadOnly        *bool
    ConcurrencySafe *bool
    MaxResultChars  int
    Timeout         time.Duration
    DeferLoad       bool
    UserFacingLabel string
}

type Tool struct {
    Name, Description string
    Parameters        map[string]any
    Handle            Handler
    Spec              ToolSpec      // 可选覆盖
    resolved          resolvedMeta  // Register 时填充
}
```

详见 [tool-spec.md](../../engineering/tool-spec.md)（M1 已实施）与 [tool-platform-overhaul.md](../../engineering/tool-platform-overhaul.md)（阶段 A～E 彻底改造）。

## 注册流程

```text
tools.RegisterAll(r, deps)
  ├── RegisterHTTPFromCatalog   // catalog.AllHTTP()
  └── RegisterBespokeTools
```

`Register()` 调用 `resolveToolMeta()`：显式 Spec + 名称推断 + `~/.geegoo/tool_policy.{yaml,json}`。

## 策略与审批

| 来源 | 行为 |
|------|------|
| 名称推断 | `create_/update_/delete_/...` → `prompt` |
| `tool_policy.yaml` | 覆盖推断（Codex execpolicy 风格） |
| `ApprovalGate` | interactive + prompt → skip 待确认 |

## 并发与结果

- `ExecuteBatch`：仅当**全部** tool 为 `ConcurrencySafe` 时并行
- `RenderResult`：按 `MaxResultChars` 截断 LLM 可见输出

## 测试

- `spec_test.go` — 推断、policy、render
- `bootstrap_test.go` — 注册数量
- `approval_test.go` — 门控
