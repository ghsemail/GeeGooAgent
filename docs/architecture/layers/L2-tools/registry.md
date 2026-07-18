# L2 — ToolRegistry

> Go 实现：`internal/tools/registry.go`、`bootstrap.go`、`bespoke.go`

## 职责

- 注册全部 Tool（当前 **82**）
- 按 toolset / chat 白名单过滤 Schema
- 导出 JSON Schema 供 LLM function calling
- 执行 Tool 并返回统一 `Result`

## 核心类型

```go
// internal/tools/registry.go

type Tool struct {
    Name        string
    Description string
    Parameters  map[string]any  // JSON Schema
    Handle      Handler
}

type Handler func(ctx Context, args map[string]any) Result

type Result struct {
    Status   Status   // ok | error | dry_run | skipped
    Summary  string
    Data     map[string]any
    ExitCode int
    Meta     map[string]any  // api_code, duration_ms, ...
}

type Registry struct { /* map[string]Tool */ }

func (r *Registry) Register(t Tool)
func (r *Registry) Schemas(filter []string) []llm.ToolSchema
func (r *Registry) Execute(req CallRequest, ctx Context) Result
```

## 注册流程

```text
tools.RegisterAll(r, deps)
  ├── RegisterHTTPFromCatalog   // catalog.AllHTTP()，排除 BespokeNames
  └── RegisterBespokeTools
        ├── registerPerceptionTools
        ├── registerAnalysisTools
        └── registerReportTools + registerMetaTools
```

`catalog.BespokeNames` 中的 Tool **不**走通用 HTTP 转发，避免双注册逻辑冲突（`search_code` 在 catalog 有定义但由 bespoke 覆盖）。

## HTTP 转发路径

`bootstrap.go` 中每个 `HTTPSpec` 包装为：

1. `ApprovalGate(spec.Name, handler)` — 写操作门控
2. `buildHTTPBody` — 支持 `MergePayload`（Bot CRUD）
3. `catalog.NeedsMCPToken` → 注入 `mcp_token`
4. `deps.HTTP.ForTool(name).Post` 或 `PostDirect`
5. `ClassifyHTTPPayload` — 空 data → skipped

## Toolset 过滤

Chat 白名单：`ChatToolNamesForToolsets(ids)`（`domains.go` + `toolset.go`）

```go
// 默认：5 个 ChatDefault toolset，排除 report_workflow 工具
RegisteredChatToolNamesFor(registry, nil)
```

Workflow 路径不使用 toolset 过滤——步骤在 `workflow/premarket.go` 硬编码 Tool 名。

## ApprovalGate

`create_` / `update_` / `delete_` / `switch_` 前缀：

| 条件 | 行为 |
|------|------|
| `DryRun` | dry_run |
| `Interactive && !Approved` | skipped + 提示确认 |
| workflow / 非 interactive | 正常执行 |

## 与 Python 蓝图差异

早期设计的 `perceive.py` / `analyze.py` 等分文件 → 现为：

| 原规划 | 现实现 |
|--------|--------|
| `perceive.py` | `bespoke.go` perception 段 + catalog HTTP |
| `analyze.py` | `bespoke.go` analysis 段 + catalog |
| `act_*.py` | catalog HTTP（Bot/报告 CRUD） |
| `meta.py` | `bespoke.go` meta + `approval.go` |

## MVP vs 当前

| 指标 | MVP 目标 | 当前 |
|------|----------|------|
| Tool 数 | ~19（盘前） | 82 注册 |
| 盘前子集 | manifest.yaml 列出 | workflow 硬编码 |
| Bash | 禁止 | 禁止 |

全量设计目录：[tool-catalog.md](./tool-catalog.md)  
运行态可用性：[tools-status.md](./tools-status.md)

## 测试

- `bootstrap_test.go` — 注册数量、dry-run 全工具
- `approval_test.go` — 门控
- `contract_test.go` — 空成功分类
- `fixture_replay_test.go` — HTTP 回放
