# Agent Loop — 功能验收

> 更新：2026-07-20。适用于本机或服务器（`~/.geegoo/bin/geegoo`）。

## 文档概述

本文档说明如何**验证 Agent Loop 行为是否符合预期**：离线 parity 卡片、Plan 门控、NDJSON、HTTP clarify、Hooks、压缩血缘及部署后检查清单。面向发布前自测、CI 与线上运维；实现细节见 [agent-loop.md](./agent-loop.md)，不要求通读代码即可按步骤验收。

## 1. 一键离线验收（推荐先做）

```bash
# 无需 config.json — stub registry + parity 卡片（12 项）
geegoo verify agent-loop --offline

# 含真实 Registry + config（需 ~/.geegoo/config.json）
export GEEGOO_CONFIG=~/.geegoo/config.json
geegoo verify agent-loop

# 配置 + loop 参数 + 内嵌 parity
geegoo inspect --quick

# 连通性 + 抽样 tool 探针
geegoo doctor
```

**期望**：`verify agent-loop` 全部 `PASS`（退出码 0）；`doctor` 全绿或仅非交易时段 WARN。

离线卡片实现：`internal/verify/agent_loop.go`、`internal/verify/offline_registry.go`。

---

## 2. 功能对照表

| 功能 | 怎么验 | 期望 |
|------|--------|------|
| 离线 parity | `geegoo verify agent-loop --offline` | 12/12 PASS，无网络 |
| `geegoo inspect` | `geegoo inspect` | profile、loop 参数、toolsets |
| NDJSON 事件流 | 见 §3 | `schema_version: 1`，`turn_complete` |
| Plan 门控 | 见 §4 | 写操作先挂起，`y` 后执行 |
| HTTP clarify | 见 §5 | SSE `clarify` + POST 续跑 |
| `delegate_tasks` | 见 §6 | 并行子任务 |
| Schema 校验 | 见 §7 | 缺参 `参数校验失败` |
| Hooks | 见 §8 | 审计脚本 stdin JSON |
| 压缩血缘 | 见 §9 | `/session` 可见 chain |
| Evaluator 重试 | `eval_max_retries: 1` + Advisor | `eval_retry` 事件 |

---

## 3. NDJSON（Headless / CI）

```bash
geegoo chat --message "腾讯代码是多少" --output-format ndjson --cli 2>/dev/null | head
```

**期望**：每行合法 JSON；含 `"schema_version":1`；末行 `turn_complete` 且 `assistant_text` 非空。

```bash
go test ./internal/cli/chatrepl/... -run NDJSON -v
```

---

## 4. Plan 门控

**前提**：`plan_gate: true`（默认）。

```bash
geegoo chat
```

触发 mutating 工具（如 `create_dca_bot`）时期望：`plan_proposed` → 横幅确认 → `y` 执行 / `n` 取消。

```bash
go test ./internal/agent/... -run PlanGate -v
go test ./internal/runtimeapi/... -run PlanHTTP -v
```

HTTP：`POST /v1/chat/plan` — 详见 [runtime-clarify.md](./runtime-clarify.md)。

---

## 5. HTTP clarify

```bash
go test ./internal/runtimeapi/... -run ClarifyHTTPE2E -v
```

协议：[runtime-clarify.md](./runtime-clarify.md)

---

## 6. 并行子 Agent

```json
{ "delegate_max_parallel": 3 }
```

离线卡片含 `delegate_tasks registered`。

---

## 7. Schema 校验

```bash
go test ./internal/tools/... -run ValidateArguments -v
```

---

## 8. Hooks

见 `scripts/hooks/audit-tool.example.sh` 与 `config.json` 的 `hooks` 段。

```bash
go test ./internal/tools/... -run HookRunner -v
```

---

## 9. 压缩血缘

```
/session
```

或 `geegoo inspect --session <id>`。

```bash
go test ./internal/agent/... -run Compress -v
```

---

## 10. 服务器验收（部署后）

```bash
cd ~/.geegoo/geegoo-agent && git log -1 --oneline
geegoo verify agent-loop --offline
geegoo doctor
```

当前基线（2026-07-20）：`20471ef` 起，offline verify **12/12 PASS**。

---

## 11. 相关文档

- [agent-loop.md](./agent-loop.md)
- [../../benchmark/agent-loop/optimization-roadmap.md](../../benchmark/agent-loop/optimization-roadmap.md)
