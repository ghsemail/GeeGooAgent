# Agent Runtime 工程边界

> 依据 [agent-runtime-architecture.md](../architecture/agent-runtime-architecture.md) §3–§5。  
> CI 通过 `go run scripts/check_import_boundaries.go` 自动校验 import 方向。

## 语言与 Ownership

| 组件 | 语言 | 规则 |
|------|------|------|
| ReAct Loop / Tool 执行 / Session 写入 | Go | **唯一** SSOT |
| Model Policy + LLM Gateway | Go | 出站模型必经此路径 |
| Cognition 默认实现 | Go | Ranker / Evaluator / PlanPolicy |
| Python Advisor | Python（可选） | **suggestion-only**；默认不部署 |
| CLI / TUI / HTTP 客户端 | Go | 无 agent 逻辑所有权 |

## Python 禁止清单

以下行为 **禁止** 出现在 Python Advisor 或任何 sidecar 中：

1. 拥有或驱动 ReAct loop / workflow 状态机  
2. 直接发起 `tool_calls` 或执行 MCP / sandbox  
3. 写入 SQLite session、checkpoint、working state  
4. 绕过 Go LLM Gateway 直连 model provider（默认路径）  
5. 返回带 `tool_calls`、`workflow_decision`、`session_write` 等字段的 JSON  

Advisor 仅允许：`rank` / `evaluate` 类建议；Go Kernel 决定是否采纳；失败必须降级。

## Go 包依赖方向

```text
cmd/* → internal/app → agent | cognition | tools | llm | memport | memory | ...
agent (Kernel) → cognition | runtime | tools | llm | memport | prompt
cognition  ↛ agent | cli | runtimeapi | tools | app
tools      ↛ cognition
memport    ↛ memory | tools | agent
infra      ↛ runtime | tools | llm | agent
```

Recall 排序：`memory.Adapter` 经 `SessionRanker` 回调 `agent.RankRecallHits` → `cognition.Ranker`，**不**在 `tools` 包 import `cognition`。

## 验收命令

```bash
go test ./internal/archboundaries/...
go run scripts/check_import_boundaries.go
go test ./internal/agent/... ./internal/cognition/... ./internal/memory/... ./internal/llm/...
go build ./cmd/geegoo ./cmd/agent-runtime
```

## 相关文档

- [agent-runtime-architecture.md](../architecture/agent-runtime-architecture.md)  
- [backlog.md](../architecture/backlog.md)  
- [repo-layout.md](../architecture/repo-layout.md)
