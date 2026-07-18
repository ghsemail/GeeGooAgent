# L2 — Tool & MCP Layer

Agent 与 GeeGoo 生态的**唯一接口**。Runtime 不手写 HTTP；所有外部 IO 经 Tool Registry。

> Go 实现：`internal/tools/` · `internal/clients/mcp/`

## 文档分工（2026-07 整理后）

| 文档 | 读什么 |
|------|--------|
| **[tools-status.md](./tools-status.md)** | **★ 运行态 SSOT**：状态、端口、韧性、踩坑、Toolset、树形总览、对话场景 |
| [tool-catalog.md](./tool-catalog.md) | **设计全集**：Phase/MVP、API 路径、Skill 子集、参数校验规则 |
| [tool-server-mapping.md](./tool-server-mapping.md) | **部署对照**：生产 IP、config 键、全量 Tool→HTTP 路径表 |
| [toolsets.md](./toolsets.md) | Toolset 分组、扩展指南 |
| [registry.md](./registry.md) | Registry API、注册流程、ApprovalGate |
| [clients.md](./clients.md) | MCP 客户端、鉴权、mcp_token |
| [sandbox-integration.md](./sandbox-integration.md) | Executor 沙箱集成 |

**外部 MCP HTTP SSOT**（73 路由）：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)

实现状态摘要 → [implementation-status.md](../../implementation-status.md) §Tool。

## 快速数字（与 `bootstrap_test` / `toolset_test` 对齐）

| 指标 | 值 |
|------|-----|
| 已注册 | **82** |
| HTTP 转发（`AllHTTP`） | **61** |
| Bespoke（`BespokeNames`） | **21** |
| Toolset 并集 | **82**（7 个 toolset，含 `prompt_template`） |
| 默认 chat 白名单 | **69** |
| workflow 独占（默认不进 chat） | **7** |

## 架构要点

```text
Agent / workflow.Runner
    → tools.Registry.Execute
        ├── bespoke.go（本地 / 直连 Signal）
        └── bootstrap.go → GeeGooBot :3120 / Signal :3200/3210/3230
```

| 决策 | 说明 |
|------|------|
| 中央 Registry | toolset 按场景过滤；workflow 步骤在 `internal/workflow/` |
| 无 Bash | 副作用走白名单 Tool |
| 禁止转发 Trading Python | 仅 GeeGoo Go 3xxx 栈 |

## 与 L5 Skill 的分工

- **Tool**：LLM function calling 或 workflow 单步 API
- **Skill**：`geegoo run` 多步任务包 → [L5 skills](../L5-application/skills.md)

## 代码包

```text
internal/tools/
├── registry.go, bootstrap.go, bespoke.go, resilience.go
├── approval.go, contract.go, toolset.go, domains.go
├── httpbackend.go
├── catalog/          # AllHTTP(), NeedsMCPToken
└── newsrunner/       # fetch_*_news Go 回退
internal/clients/mcp/
internal/search/      # web_search
```

## 维护

新增 Tool 后同步：**`tools-status.md`** + **`tool-catalog.md`** + `internal/tools/catalog/catalog.go` + [interface-map.md](../../../reference/geegoo-mcp/interface-map.md)（新 HTTP 时）。

```bash
go test ./internal/tools/... -count=1
```
