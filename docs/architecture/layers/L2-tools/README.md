# L2 — Tool & MCP Layer

Agent 与 GeeGoo 生态的**唯一接口**。Runtime 不手写 HTTP；所有外部 IO 经 Tool Registry。

> Go 实现：`internal/tools/` · `internal/clients/mcp/`

## 文档索引（本层 SSOT）

实现状态摘要 → [implementation-status.md](../../implementation-status.md) §Tool。

| 文档 | 读什么 |
|------|--------|
| **[tools-reference.md](./tools-reference.md)** | **★ 全量 82 Tool**：名称、作用、来源服务器、HTTP 接口、实现状态、问题 |
| [tools-tree.md](./tools-tree.md) | 树形总览、chat 场景、踩坑速查 |
| [tool-catalog.md](./tool-catalog.md) | Phase/MVP、参数校验规则 |
| [toolsets.md](./toolsets.md) | Toolset 分组、五类 taxonomy、扩展指南 |
| [registry.md](./registry.md) | Registry API、注册流程、ApprovalGate |
| [clients.md](./clients.md) | MCP 客户端、鉴权、mcp_token |
| [tool-server-mapping.md](./tool-server-mapping.md) | Tool → 服务器 IP/端口/HTTP 路径 |

**外部 MCP HTTP SSOT**（73 路由）：[interface-map.md](../../../reference/geegoo-mcp/interface-map.md)

## 快速数字

| 指标 | 值 |
|------|-----|
| 已注册 | **82** |
| HTTP 转发 | 62 |
| Bespoke | 21 |
| 默认 chat 白名单 | ~73 |

## 架构要点

```text
Agent / workflow.Runner
    → tools.Registry.Execute
        ├── bespoke.go（本地 / 直连 Signal）
        └── bootstrap.go → GeeGooBot :3120 / Signal :3200/3210
```

| 决策 | 说明 |
|------|------|
| 中央 Registry | toolset 按场景过滤；workflow 硬编码步骤 |
| 无 Bash | 副作用走白名单 Tool |
| 禁止转发 Trading Python | 仅 GeeGoo Go 3xxx 栈 |

## 与 L5 Skill 的分工

- **Tool**：LLM function calling 或 workflow 单步 API
- **Skill**：`geegoo run` 多步任务包 → [L5 skills](../L5-application/skills.md)

## 代码包

```text
internal/tools/
├── registry.go, bootstrap.go, bespoke.go
├── approval.go, contract.go, toolset.go, domains.go
├── httpbackend.go
└── catalog/          # AllHTTP(), NeedsMCPToken
internal/clients/mcp/ # GeeGooBot HTTP 客户端
internal/search/      # web_search
```

## 边界

- **提供**：Tool 注册、执行、Schema、HTTP 客户端
- **不提供**：ReAct 循环（L4）、Skill 定义（L5）、SQLite（L0）
