# GeeGoo 平台总览

> **本文定位**：多仓库平台的**唯一入口**——说明各服务如何分工、文档在哪里、新人该读什么。  
> **GeeGooAgent 仓库**承载 Agent Runtime 与大部分架构文档；各服务仓库保留各自的 API / 部署 SSOT。

---

## 1. 平台是什么

GeeGoo 是一套 **自托管 Agent + 运营门户 + 后端服务** 的组合：

```text
┌─────────────────────────────────────────────────────────────────────────┐
│  用户 / 运营                                                             │
└───────────────────────────────┬─────────────────────────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        ▼                       ▼                       ▼
 trading_operation          geegoo CLI            飞书 / 其他 Gateway
 (Flutter Web 门户)         (调试 TUI)              (可选)
        │                       │
        └───────────┬───────────┘
                    ▼
         GeeGooAgent agent-runtime :3400
         ReAct Loop · Session · Eval · Cockpit API
                    │
     ┌──────────────┼──────────────┬──────────────┐
     ▼              ▼              ▼              ▼
 GeeGooBot      GeeGooData    GeeGooSignal   (外部 LLM)
 mcp-api :3120  :3300         :3210
 agent-api :3110
```

**设计公式**（与 [platform-blueprint](../architecture/platform-blueprint/README.md) 一致）：

```text
Agent = L4 Runtime + L0 Infrastructure + L5 Skill Pack
```

GeeGoo 实例上，L2 Tool 通过 MCP HTTP 对接 GeeGooBot；行情与报告走 GeeGooData；策略生成走 GeeGooSignal。

---

## 2. 仓库地图

| 仓库 | 角色 | 默认端口 | 文档入口 |
|------|------|----------|----------|
| **[GeeGooAgent](https://github.com/ghsemail/GeeGooAgent)** | Agent OS：`geegoo` CLI、`agent-runtime`、Eval、架构 SSOT | `3400` | [docs/README.md](../README.md) |
| **[GeeGooBot](https://github.com/ghsemail/GeeGooBot)** | Go 后端：MCP / App / Agent BFF / Worker | `3100–3140`, `3120` MCP | [docs/README.md](https://github.com/ghsemail/GeeGooBot/blob/main/docs/README.md) |
| **[GeeGooData](https://github.com/ghsemail/GeeGooData)** | 行情、资金、报告数据服务 | `3300` | [docs/README.md](https://github.com/ghsemail/GeeGooData/blob/main/docs/README.md) |
| **[GeeGooSignal](https://github.com/ghsemail/GeeGooSignal)** | 信号与策略生成 | `3210` | [docs/README.md](https://github.com/ghsemail/GeeGooSignal/blob/main/docs/README.md) |
| **[trading_operation](https://github.com/ghsemail/trading_operation)** | Flutter Web 运营门户（Chat + Cockpit + Eval UI） | Nginx 静态 | 见 [dashboard-platform.md](../architecture/dashboard-platform.md) |
| **[trading_app](https://github.com/ghsemail/trading_app)** | 移动端（Flutter） | — | 仓库内 PRD 片段为主 |
| **[TradingServer](https://github.com/ghsemail/TradingServer)** | 遗留 Futu 交易服务（Python） | — | [README](https://github.com/ghsemail/TradingServer/blob/main/README.md)（历史部署） |

**API 文档 SSOT 分工**

| 主题 | 权威文档位置 |
|------|----------------|
| MCP 路由 × Agent Tool 映射 | GeeGooAgent [reference/geegoo-mcp/interface-map.md](../reference/geegoo-mcp/interface-map.md) |
| MCP HTTP 已实现路由 | GeeGooBot [api/implemented-routes.md](https://github.com/ghsemail/GeeGooBot/blob/main/docs/api/implemented-routes.md) |
| Agent Runtime HTTP | GeeGooAgent [api/](../api/) · [entrypoints.md](../architecture/entrypoints.md) |
| Tool 注册与运行态 | GeeGooAgent [tools-status.md](../architecture/layers/L2-tools/tools-status.md) |

---

## 3. 文档阅读路径

按目标选入口，不要从根目录随机点开 markdown。

| 你是谁 | 先读 | 再读 |
|--------|------|------|
| **新加入的 Agent 开发者** | [architecture/overview.md](../architecture/overview.md) | [implementation-status.md](../architecture/implementation-status.md) → [agent-loop.md](../architecture/layers/L4-runtime/agent-loop.md) |
| **要 fork 新领域 Agent** | [platform-blueprint/README.md](../architecture/platform-blueprint/README.md) | [agent-build-guide.md](../architecture/platform-blueprint/agent-build-guide.md) → [eval-build-guide.md](../architecture/platform-blueprint/eval-build-guide.md) |
| **做 Eval / 自动化验收** | [eval-build-guide.md](../architecture/platform-blueprint/eval-build-guide.md)（机制） | [turnplan-eval.md](../engineering/turnplan-eval.md)（本仓库套件） |
| **改 Tool / MCP 对接** | [interface-map.md](../reference/geegoo-mcp/interface-map.md) | [tools-status.md](../architecture/layers/L2-tools/tools-status.md) · [tool-spec.md](../engineering/tool-spec.md) |
| **做 Web / Cockpit** | [dashboard-platform.md](../architecture/dashboard-platform.md) | [deploy/web-platform.md](../../deploy/web-platform.md) |
| **运维部署** | [cross-cutting/deployment.md](../architecture/cross-cutting/deployment.md) | 各服务仓库 `docs/deployment.md` |
| **查实现进度** | [implementation-status.md](../architecture/implementation-status.md) | [backlog.md](../architecture/backlog.md)（唯一待办） |

完整索引：[docs/README.md](../README.md)。

---

## 4. GeeGooAgent 文档分区

```text
docs/
├── README.md                 # 本仓库文档总索引
├── platform/                 # ★ 多仓库平台（本文）
├── architecture/             # 六层架构 SSOT
│   ├── README.md
│   ├── implementation-status.md / backlog.md
│   ├── gateway/              # 飞书等入口
│   ├── dashboard-platform.md # Web 三仓分工
│   └── platform-blueprint/   # 通用 Agent 蓝图
├── engineering/              # 编码规范、Eval 操作、Cursor 工作流
├── reference/geegoo-mcp/     # MCP 专题参考（73 路由）
├── benchmark/                # Hermes / Grok / Codex 对标
├── api/                      # Runtime HTTP 事件等
├── superpowers/              # 进行中的跨仓设计稿（带日期）
└── archive/                  # 历史计划（只读）
```

**命名约定**：GeeGooAgent / GeeGooBot 用小写文件名（`architecture.md`）；GeeGooData / GeeGooSignal 用大写（`ARCHITECTURE.md`）。各仓库内以该仓库 `docs/README.md` 为准。

---

## 5. 运行时端口（31xx / 33xx / 34xx）

| 服务 | 端口 | 说明 |
|------|------|------|
| GeeGooBot app-api | 3100 | App API |
| GeeGooBot agent-api | 3110 | Agent BFF（可选） |
| GeeGooBot mcp-api | 3120 | MCP Tool HTTP |
| GeeGooBot service-api | 3140 | 内部服务 API |
| GeeGooSignal | 3210 | 策略 / 信号 |
| GeeGooData | 3300 | 行情与报告数据 |
| GeeGooAgent agent-runtime | 3400 | Chat SSE、Eval、Cockpit |

`geegoo setup` 写入 `~/.geegoo/config.json` 默认值。详见根目录 [README.md](../../README.md#runtime-ports)。

---

## 6. 平台健康度速览（文档视角）

| 维度 | 状态 | 说明 |
|------|------|------|
| Agent 架构与 Loop | ✅ 成熟 | `architecture/` + `implementation-status` 维护良好 |
| MCP / Tool 映射 | ✅ | `interface-map.md` 为 SSOT；专题在 `reference/geegoo-mcp/` |
| Eval 机制 | ✅ | 通用 [eval-build-guide](../architecture/platform-blueprint/eval-build-guide.md) + 领域 [turnplan-eval](../engineering/turnplan-eval.md) |
| Web 门户 | ✅ 已定稿 | `dashboard-platform.md`；实现分散在 trading_operation |
| 跨仓链接 | ⚠️ 持续治理 | 不用 `../../OtherRepo/` 相对路径；用本文 §2 的 GitHub 链接 |
| 历史稿 | 📦 已归档 | `PROGRESS.md`（Python 时代）→ 见 [implementation-status](../architecture/implementation-status.md) |
| Flutter 应用文档 | ⚠️ 偏薄 | trading_operation / trading_app 缺独立 docs；以 Agent 侧 Cockpit 文档为准 |

---

## 7. 维护规则

1. **实现状态**只写在 [implementation-status.md](../architecture/implementation-status.md)；**待办**只写在 [backlog.md](../architecture/backlog.md)。
2. **Tool 是否可调用**以 [tools-status.md](../architecture/layers/L2-tools/tools-status.md) 为准，不在 overview 里重复列全表。
3. **MCP 变更**先改 `interface-map.md`，再同步 Tool 注册与 `reference/geegoo-mcp/` 专题。
4. **Eval 机制**改 [eval-build-guide.md](../architecture/platform-blueprint/eval-build-guide.md)；**本仓库用例**改 `internal/eval/` + [turnplan-eval.md](../engineering/turnplan-eval.md)。
5. **跨仓库引用**用 GitHub 绝对 URL 或本文 §2 表格，避免 monorepo 式相对路径。
