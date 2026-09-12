# GeeGooAgent 文档

> **多仓库平台入口**：[platform/README.md](./platform/README.md)（各服务分工、端口、跨仓文档地图）  
> **本仓库架构 SSOT**：[architecture/README.md](./architecture/README.md)

---

## 快速导航

| 目标 | 文档 |
|------|------|
| 第一次读代码库 | [architecture/overview.md](./architecture/overview.md) |
| Agent Runtime 定稿 | [architecture/agent-runtime-architecture.md](./architecture/agent-runtime-architecture.md) |
| **已实现 / 未实现** | [architecture/implementation-status.md](./architecture/implementation-status.md) |
| **待办（唯一）** | [architecture/backlog.md](./architecture/backlog.md) |
| Tool 运行态 | [architecture/layers/L2-tools/tools-status.md](./architecture/layers/L2-tools/tools-status.md) |
| Agent Loop | [architecture/layers/L4-runtime/agent-loop.md](./architecture/layers/L4-runtime/agent-loop.md) |
| Loop 验收命令 | [architecture/layers/L4-runtime/agent-loop-verification.md](./architecture/layers/L4-runtime/agent-loop-verification.md) |
| Web 门户三仓分工 | [architecture/dashboard-platform.md](./architecture/dashboard-platform.md) |
| Fork 新 Agent | [architecture/platform-blueprint/README.md](./architecture/platform-blueprint/README.md) |
| Eval 机制（通用） | [architecture/platform-blueprint/eval-build-guide.md](./architecture/platform-blueprint/eval-build-guide.md) |
| Eval 套件（本仓库） | [engineering/turnplan-eval.md](./engineering/turnplan-eval.md) |

---

## 文档分区

### 平台与架构

| 目录 | 用途 |
|------|------|
| [platform/](./platform/) | 多仓库平台总览、仓库地图、阅读路径 |
| [architecture/](./architecture/) | 六层架构、实现状态、领域映射、Gateway、Dashboard |
| [architecture/platform-blueprint/](./architecture/platform-blueprint/) | 通用自托管 Agent 蓝图（fork 新领域用） |
| [architecture/gateway/](./architecture/gateway/) | 飞书等外部入口 |
| [architecture/cross-cutting/](./architecture/cross-cutting/) | 部署、可观测性、Supervisor |

### 工程与 API

| 目录 | 用途 |
|------|------|
| [engineering/](./engineering/) | 需求、编码/测试规范、Cursor 工作流、Agent OS 边界 |
| [reference/geegoo-mcp/](./reference/geegoo-mcp/) | MCP HTTP 参考（**interface-map** 为 Tool 映射 SSOT） |
| [api/](./api/) | Agent Runtime HTTP（MCP serve、runtime events 等） |

### Eval

| 文档 | 用途 |
|------|------|
| [eval-build-guide.md](./architecture/platform-blueprint/eval-build-guide.md) | 领域无关：Plan-only / Live / Verify / Job |
| [turnplan-eval.md](./engineering/turnplan-eval.md) | 本仓库 TurnPlan 套件与操作 |
| [scripts/eval/README.md](../scripts/eval/README.md) | 生成器、迁移、冒烟脚本 |

### 对标与历史

| 目录 | 用途 |
|------|------|
| [benchmark/](./benchmark/) | vs Hermes / Grok / Codex |
| [superpowers/](./superpowers/) | 进行中的跨仓设计稿（带日期） |
| [archive/](./archive/) | 历史计划与 Superpowers 归档（非架构正文） |

### 仓库外（运维）

| 路径 | 用途 |
|------|------|
| [deploy/](../deploy/) | Hermes 对齐、Web 平台、systemd |
| 根目录 [README.md](../README.md) | Quick Start、CLI、端口 |

---

## 六层文档

均在 `architecture/layers/`：

| 层 | 目录 |
|----|------|
| L0 Infrastructure | [L0-infrastructure/](./architecture/layers/L0-infrastructure/) |
| L1 Model Gateway | [L1-model-gateway/](./architecture/layers/L1-model-gateway/) |
| L2 Tools | [L2-tools/](./architecture/layers/L2-tools/) |
| L3 Memory | [L3-memory/](./architecture/layers/L3-memory/) |
| L4 Runtime | [L4-runtime/](./architecture/layers/L4-runtime/) |
| L5 Application | [L5-application/](./architecture/layers/L5-application/) |

---

## 过时文档

| 文件 | 说明 |
|------|------|
| [../PROGRESS.md](../PROGRESS.md) | **已废弃**（Python/pytest 时代 Step 0–15）。以 [implementation-status.md](./architecture/implementation-status.md) 为准。 |
