# ToolSpec 统一改造方案

> 状态：已实施（2026-08-31）  
> 目标：将散落在 approval / timeouts / metadata / loop 的工具元数据收拢为单一 `ToolSpec`，供 Registry、Loop、MCP、运营台共用。

## 背景

此前 `tools.Tool` 仅含 `Name/Description/Parameters/Handle`，以下能力靠命名约定与分散 map 维护：

| 能力 | 旧位置 |
|------|--------|
| 写操作审批 | `approval.go` 前缀 `create_/update_/...` |
| 超时 | `timeouts.go` 独立 map |
| 结果截断 | `loop_reply.go` 全局 6000 字符 |
| 并发 | `tool_exec.go`「非 mutating 即并行」 |
| 运营台元数据 | `metadata.go` 反查名字 |

## 设计

### 核心类型（`internal/tools/spec.go`）

```go
type PolicyAction string // allow | prompt | forbidden

type ToolSpec struct {
    Policy          PolicyAction   // 空则按名称推断
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

`Register()` 调用 `resolveToolMeta()`：显式 Spec 覆盖 + 名称推断 + `tool_policy.yaml` 规则。

### 策略文件（`~/.geegoo/tool_policy.yaml`）

```yaml
rules:
  - match: "delete_*"
    action: prompt   # 可改为 forbidden 收紧生产
  - match: "get_*|list_*|search_*"
    action: allow
```

加载：`tools.LoadPolicyFile(path)`，在 `app` 启动时调用（文件不存在则跳过）。

### 消费方

| 模块 | 改动 |
|------|------|
| `approval.go` | `ApprovalRequired` → `resolved.Policy == prompt` |
| `timeouts.go` | `ExecutionTimeout` → 读 `resolved.Timeout` |
| `metadata.go` | `RequiresApproval` / 未来 `ReadOnly` 来自 resolved |
| `tool_exec.go` | 并行条件：`all calls ConcurrencySafe` |
| `loop_plan.go` | `RenderResultForLLM(name, result, registry)` |
| `inspect` | 展示 prompt/forbidden/defer 工具计数 |
| `verify` | 新增 ToolSpec 一致性卡片 |

### 推断规则（默认，可被 Spec / YAML 覆盖）

- **Policy prompt**：`create_/update_/delete_/edit_/switch_/add_` 前缀
- **ReadOnly**：`policy != prompt` 且非 `write_/save_` 前缀
- **ConcurrencySafe**：`get_/list_/search_/check_/fetch_` 前缀；`delegate_*`、`clarify`、写操作为 false
- **MaxResultChars**：默认 6000；`list_*` 4000；`get_strategy_backtest_log` 8000

### 兼容性

- 所有现有 `Register(Tool{...})` 无需改动，推断在 `Register` 时自动完成
- `Schemas()` 默认仍包含全部工具（`DeferLoad` 仅元数据标记，下阶段接 `discover_tools`）
- 行为与改前一致，除非显式配置 `tool_policy.yaml`

**完整彻底改造路线图（阶段 A～E）：** [tool-platform-overhaul.md](./tool-platform-overhaul.md)

## 验收

```bash
go test ./internal/tools/... ./internal/agent/...
go run ./cmd/geegoo verify agent-loop --offline
geegoo inspect   # Tools 段含 PromptTools / DeferLoad 计数
```

## 后续（可选加深）

- 从 `HTTPSpec` / OpenAPI codegen `Parameters`（阶段 E 深化）
- 21 个 bespoke struct 参数化
- 语义 embedding ToolSearch（阶段 C+）
