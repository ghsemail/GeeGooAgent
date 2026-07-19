# Agent Loop 优化方案

> 基于 [Hermes 对标](./hermes.md)、[Grok Build 对标](./grok-build.md) 与 GeeGoo 现有实现整理。  
> 更新：2026-07-19。原则：**强化金融场景优势，借鉴 harness 能力，不照搬编码 Agent 工具链**。

## 0. 已上线（2026-07-19）

| 提交 | 主机 | 验收 |
|------|------|------|
| `225615ed` | 119.45.16.112（`~/.geegoo/geegoo-agent`） | `geegoo doctor` 全绿；`geegoo inspect --quick` 9/9 PASS |

**本批交付**：`geegoo inspect`、`--output-format ndjson`、HTTP clarify、`plan_gate`、工具 schema 校验、Hooks、`delegate_tasks`、`delegate_max_parallel`。

## 1. 目标与边界

### 1.1 优化目标

| 目标 | 说明 |
|------|------|
| **Loop 可预测** | 复杂写操作先计划、后执行；预算/压缩/中断行为可验收 |
| **Loop 可集成** | Headless、HTTP、scheduler 共用同一套事件与进度协议 |
| **Loop 可扩展** | 工具契约、Deps、Hooks 松耦合，少改核心循环 |
| **Loop 可并行** | 多标的 / 多子任务在 MCP 限流内安全并行 |
| **保持差异化** | Workflow + Supervisor + Evidence + verify **不削弱** |

### 1.2 明确不做（对标项目有、GeeGoo 不采纳）

| 能力 | 来源 | 原因 |
|------|------|------|
| 任意 Shell / 终端 backend | Hermes / Grok | 非 coding agent；安全风险 |
| 文件读写 / patch / Git | Grok | 与 MCP 金融域无关 |
| 18+ Provider / anthropic_messages | Hermes | 按需精简，维持 3 家 OpenAI 兼容 |
| IM Gateway / ACP | Hermes / Grok | 无产品需求 |
| 向量记忆 / Context Engine 插件 | Hermes | 单租户 YAGNI |
| 轨迹训练导出 ShareGPT | Hermes | 无训练管线 |

---

## 2. 现状快照（Loop 相关）

| 能力 | 状态 | 代码 / 命令 |
|------|------|-------------|
| ReAct 主循环 | ✅ | `internal/agent/loop.go` |
| 压缩 + hygiene | ✅ | `internal/prompt/compressor.go` |
| Prompt cache 断点 | ✅ | `internal/llm/cache.go` |
| clarify + TUI | ✅ | `internal/tools/clarify.go`, `chattui` |
| clarify + HTTP runtime | ✅ | `internal/runtimeapi/clarify_hub.go` |
| delegate_task | ✅ | `internal/agent/subagent.go` |
| delegate_tasks | ✅ | 并行子 Agent，`delegate_max_parallel` |
| 工具 schema 校验 | ✅ | `internal/tools/schema_validate.go` |
| Headless 对话 `-p` | ❌ | 仅有 `run` / `verify` / scheduler |
| Plan 批准门控 | ✅ | `plan_gate` + `plan_proposed` 事件 |
| 并行子 Agent | ✅ | `delegate_tasks` + `delegate_max_parallel` |
| Hooks | ✅ | `config.hooks` + `HookRunner` |
| `geegoo inspect` | ✅ | `cmd/geegoo/inspect.go` |
| Cost / token 预算 | ❌ | 无 `internal/llm/cost` |
| 会话压缩血缘 | ⚠️ | metadata 有，未完整 FTS 血缘 |
| Deps 松耦合 | ⚠️ | MCP/Search 硬编进 `app.App` |

> 注：`docs/architecture/layers/L5-application/subagents.md` 与 `implementation-status.md` 中「子 Agent ❌」已过时，应以 `delegate_task` + `geegoo verify agent-loop` 为准。

---

## 3. 优化路线图总览

```text
Phase A（4–6 周）  体验与集成 — 低风险、用户可感知
Phase B（6–8 周）  可靠性与契约 — 少踩坑、好排障
Phase C（8–12 周） 吞吐与扩展 — 并行、Hooks、松耦合
Phase D（按需）    平台化 — Cost、Webhook、辅助 LLM
```

| Phase | 主题 | 主要借鉴 | 交付物 |
|-------|------|----------|--------|
| **A** | 可集成 + 可发现 | Grok Headless/inspect；Hermes clarify 全入口 | 统一事件流、`geegoo inspect`、runtime clarify |
| **B** | 可预测 + 可验收 | Grok Plan mode；Hermes jsonschema | Plan 门控、工具契约、文档与 verify 扩展 |
| **C** | 可扩展 + 可并行 | Grok 并行子 Agent；Hermes check_fn | 并行 delegate、Hooks、Deps 注册表 |
| **D** | 运营与成本 | Hermes auxiliary / fallback 深化 | Cost manager、Webhook、压缩血缘 |

---

## 4. Phase A — 体验与集成（优先落地）

### A1. 统一 Agent 事件协议（借鉴 Grok `streaming-json`）

**问题**：TUI、`EmitProgress`、HTTP runtime、EventBus 字段不一致，CI/脚本难以解析。

**方案**：

1. 定义稳定 JSON 事件 schema（版本字段 `schema_version: 1`）：
   - `turn_start` / `round_start` / `stream_delta` / `tool_start` / `tool_done` / `turn_complete` / `turn_failed` / `budget_exhausted`
2. `internal/runtime/progress.go` 增加 `ProgressEncoder`（text | ndjson）
3. HTTP runtime SSE 或 chunked NDJSON 复用同一 encoder
4. `geegoo chat --message "..." --output-format ndjson`（非 TTY 单次提问）

**涉及**：`internal/runtime/progress.go`、`internal/runtimeapi/`、`cmd/geegoo/chat.go`

**验收**（2026-07-19 已达成）：

- [x] 单次 chat 与 HTTP 请求产出事件类型集合一致（NDJSON `schema_version: 1`）
- [x] `geegoo chat --output-format ndjson` 可用
- [ ] 集成测试：解析 NDJSON 得到 `turn_complete`（待补 CI）

**工作量**：中（~1–2 周）

---

### A2. `geegoo inspect`（借鉴 Grok `grok inspect`）

**问题**：`doctor` 探活强，但看不到当前 profile、toolsets、skills、rules、已注册工具数。

**方案**：新增子命令 `geegoo inspect [--config PATH]`，只读汇总：

| 区块 | 内容 |
|------|------|
| Config | 解析后的 provider/model、profile、dry_run、chat_toolsets |
| Loop | max_steps、sub_agent_max_steps、compression 阈值 |
| Tools | 注册数、当前 toolset 展开列表、workflow 独占工具 |
| Skills | `geegoo skills list` 摘要 |
| Runtime | agent-runtime URL、MCP base（脱敏） |
| Verify | 上次 `verify agent-loop` 可内嵌快速跑（可选 `--quick`） |

**涉及**：`cmd/geegoo/inspect.go`（新）、复用 `app.Load`、`tools.ListByToolset`

**验收**（2026-07-19 已达成）：

- [x] 无网络也可运行（除 `--quick` verify）
- [x] 输出适合粘贴到 issue / 运维群

**工作量**：小（~3–5 天）

---

### A3. HTTP runtime 支持 clarify（借鉴 Hermes gateway 回调）

**问题**：`clarify` 在 TUI 阻塞选 A/B/C；HTTP 调用无法完成澄清，loop 卡住或 skip。

**方案**：

1. `runtimeapi` 增加 **两阶段**协议之一（二选一，推荐 ①）：
   - ① **同步**：响应 `needs_clarification` + `choices`，客户端 POST `/v1/chat/clarify` 带 `answer` 续跑同一 session
   - ② **异步**：202 + `clarification_id`，轮询或 webhook（Phase D）
2. `tools.Context.ClarifyFn` 在 runtime 注入 channel/API handler
3. OpenAPI 文档补充 clarify 状态机

**涉及**：`internal/runtimeapi/handler.go`、`internal/tools/clarify.go`

**验收**（2026-07-19 已达成）：

- [x] runtime handler + `clarify_hub` 单测通过
- [ ] E2E：HTTP 触发 clarify → 提交选项 → 得到最终回复（待补自动化）
- [ ] TUI 行为不变

**工作量**：中（~1–2 周）

---

### A4. 文档与 SSOT 修正

- 更新 `implementation-status.md`、`subagents.md`：`delegate_task` ✅
- `agent-loop.md` 补充 NDJSON / inspect / runtime clarify 链接（实施后）

**工作量**：小（1 天）

---

## 5. Phase B — 可靠性与契约

### B1. Plan 门控（借鉴 Grok Plan mode，GeeGoo 化）

**问题**：Bot 创建、批量改参等写操作，ReAct 可能在用户未确认前就调 MCP。

**方案**（**不写代码库文件**，仅编排门控）：

```text
检测到 mutating tool（已有 approval 列表）
  → 若未处于 approved 状态：
       1. LLM 仅允许输出结构化「计划」消息（或调用 plan_summary tool）
       2. TUI 展示计划 + diff 式参数摘要（非文件 diff）
       3. 用户确认后 SetApproved(true) 再执行原 tool_calls
```

可选实现路径：

| 路径 | 适用 |
|------|------|
| **Chat 内状态机** | `geegoo chat` 复杂单次任务 |
| **Workflow 步骤** | 盘前/盘中已确定性步骤（优先用现有 ApprovalGate） |

**涉及**：`internal/agent/loop_round.go`（plan 轮禁止 tool）、`chattui` 计划面板、`tools/approval.go`

**验收**：

- [ ] `create_bot` 类工具在未确认时不发出 HTTP 写请求
- [ ] dry-run 仍跳过写操作
- [ ] 单元测试：plan 轮 `tool_calls` 被剥离或拦截

**工作量**：中–大（~2–3 周）

---

### B2. 工具 Schema 契约（借鉴 Hermes jsonschema）

**问题**：参数类型错误到 MCP 才失败，浪费 loop 轮次。

**方案**：

1. 为高频 mutating / 多参工具补充 JSON Schema（或 Go struct → schema 生成）
2. `Registry.Execute` 前校验；失败返回 `StatusError` + 可读字段提示（不进 LLM 长堆栈）
3. `geegoo verify tools`（可选）抽样校验 schema 可解析

**涉及**：`internal/tools/registry.go`、`internal/tools/catalog/` 各工具

**验收**（2026-07-19 已达成）：

- [x] `Registry.Execute` 前校验必填项与基础类型
- [x] `verify agent-loop` schema 卡片 PASS
- [ ] 错误参数 1 轮内被模型纠正（抽样 E2E 待补）

**工作量**：大（持续；首批 2 周覆盖 Action 类）

---

### B3. 压缩与血缘可观测（借鉴 Hermes session lineage）

**问题**：长会话压缩后难以追溯「哪次压缩丢了什么」。

**方案**：

1. 压缩时写入 `session.Metadata.lineage`：parent_id、cut_index、summary_hash、token_before/after
2. `geegoo inspect` / TUI `/session` 展示代数
3. 摘要失败跳过压缩时写 `compression_skipped` 事件（已有逻辑，补观测）

**涉及**：`internal/agent/loop_compress.go`、`internal/chatsession/`

**验收**：

- [ ] 人工触发压缩后 inspect 可见 lineage 链
- [ ] `loop_compress_test` 覆盖 metadata

**工作量**：中（~1 周）

---

### B4. 扩展 `geegoo verify agent-loop`

在现有卡片基础上增加（实施后）：

| 检查项 | 说明 |
|--------|------|
| NDJSON encoder 注册 | 事件类型非空 |
| clarify runtime handler | handler 存在且测试通过 |
| plan gate 配置 | `agent.plan_gate` 默认 mutating 列表 |

**工作量**：小（随 A/B 增量）

---

## 6. Phase C — 吞吐与扩展

### C1. 并行 `delegate_task`（借鉴 Grok 多路子 Agent）

**问题**：多标的调研串行，耗时长。

**方案**：

1. `ToolExec` 增加 **delegate 并行池**（与 `tool_max_parallel` 分开配置，默认 2–3）
2. 父任务工具 `delegate_tasks`（批量）或允许单轮多个 `delegate_task` 并行
3. **MCP 限流**：全局 semaphore（按 `mcp_token`），避免打爆 GeeGooBot
4. 子 Agent 结果合并为结构化摘要返回父 loop

**涉及**：`internal/agent/tool_exec.go`、`internal/agent/subagent.go`、`internal/app` 限流配置

**验收**（2026-07-19 部分达成）：

- [x] 嵌套 delegate 仍拒绝（`verify agent-loop`）
- [x] `delegate_tasks` + `delegate_max_parallel` 已上线
- [ ] 2 路 delegate 总耗时 < 串行 1.7×（mock MCP 基准待补）

**工作量**：大（~2–3 周）

---

### C2. Hooks（借鉴 Grok / Hermes 插件钩子）

**问题**：合规审计、告警只能改 Go 代码。

**方案**（轻量，非完整插件系统）：

```json
"hooks": {
  "tool_before": ["scripts/audit-tool.sh"],
  "tool_after": ["scripts/audit-tool.sh"],
  "turn_complete": []
}
```

- 钩子仅接收 JSON stdin（tool 名、args 摘要、status、duration）
- 超时 5s，失败记日志 **不阻断** loop（可配置 `hooks.fail_closed`）

**涉及**：`internal/config`、`internal/agent/tool_exec.go`、文档

**验收**（2026-07-19 已达成基础版）：

- [x] `config.hooks` + `HookRunner` 接入 `Registry.Execute`
- [x] `fail_closed` 可阻断工具执行（单测）
- [ ] 配置 hook 后写操作在日志留审计行（运维文档待补）

**工作量**：中（~1–2 周）

---

### C3. Deps 注册表（借鉴 Hermes check_fn / 松耦合）

**问题**：`app.App` 硬编码 MCP client、Search，测试与替换成本高。

**方案**：

1. `internal/platform/deps.go`：`DepsRegistry` 接口（MCP、Search、News、…）
2. `app.Load` 从 config 组装默认实现；测试注入 mock
3. 工具 `Handle` 只读 `tools.Context.Deps`

**涉及**：`internal/app/app.go`、`internal/tools/context.go`、各 registrar

**验收**：

- [ ] agent 测试无需真实 MCP 即可跑通 delegate 路径
- [ ] 无行为变更（回归 `go test ./internal/agent/...`）

**工作量**：大（~2–3 周，宜分 PR）

---

### C4. 子 Agent 指定模型（借鉴 Grok per-child model）

**方案**：`delegate_task` 增加可选 `model` / `thinking`；子 `SubAgent` 临时 `SetGateway` 克隆。

**场景**：子任务用便宜模型做检索，主会话用强模型综合。

**工作量**：小–中（~1 周）

---

## 7. Phase D — 运营与成本（按需）

| 项 | 借鉴 | 方案 | 优先级 |
|----|------|------|--------|
| **Cost Manager** | Hermes token 统计深化 | 按 session/skill 累计 prompt/completion/cache hit；status bar + inspect | 中 |
| **Webhook 触发** | Hermes gateway webhook | `POST /hooks/run-skill` 鉴权触发 `RunSkill` | 低 |
| **辅助 LLM** | Hermes `auxiliary_client` | 压缩摘要、证据摘要走小模型，主 loop 不变 | 低 |
| **Fallback 策略可视化** | Hermes fallback_providers | inspect 展示 fallback 链与上次失败原因 | 低 |

---

## 8. 与 Workflow 的关系（避免重复建设）

| 能力 | Chat ReAct | Workflow | 建议 |
|------|------------|----------|------|
| 盘前定时 | ❌ 不用 | ✅ scheduler | 保持 |
| 用户问一句 | ✅ chat | — | 保持 |
| 写操作审批 | Phase B Plan 门控 | 已有 ApprovalGate | Workflow 继续用步骤级；chat 用 Plan 门控 |
| 并行多股 | Phase C delegate | 已有 per-stock 步骤 | 盘前用 Workflow；chat 调研用并行 delegate |
| 验收 | verify 业务字段 | Supervisor | 两套互补 |

**原则**：不把盘前主链路改成纯 ReAct；Loop 优化服务于 **chat 按需分析** 与 **HTTP 集成**。

---

## 9. 优先级矩阵（决策用）

|  | 用户价值 | 实现成本 | 建议 Phase |
|--|----------|----------|------------|
| 统一 NDJSON 事件 | 高 | 中 | **A1** |
| geegoo inspect | 高 | 低 | **A2** |
| runtime clarify | 高 | 中 | **A3** |
| Plan 门控 | 高 | 中–大 | **B1** |
| 工具 schema | 高 | 大 | **B2**（分批） |
| 并行 delegate | 中 | 大 | **C1** |
| Hooks | 中 | 中 | **C2** |
| Deps 注册表 | 中（维护性） | 大 | **C3** |
| Cost manager | 中 | 中 | **D** |

---

## 10. 建议执行顺序（接下来 8 周）

```text
Week 1–2   A2 inspect + A4 文档修正
Week 2–4   A1 NDJSON 事件协议 + chat --output-format
Week 4–5   A3 HTTP clarify
Week 5–7   B1 Plan 门控（chat mutating tools）
Week 7–8   B3 压缩 lineage + B4 verify 扩展
           （并行启动 B2 schema 首批 Action 工具）
Week 9+    C1 并行 delegate 或 C2 Hooks（按业务痛点二选一）
```

---

## 11. 成功指标

| 指标 | 基线 | 目标 |
|------|------|------|
| HTTP 集成 clarify 成功率 | 0% | ≥95% 多选场景 |
| mutating 误执行（未确认） | 偶发 | 0（Plan 门控后） |
| 多标的 chat 调研耗时 | 串行 N×T | 并行 ≤0.6N×T（N≤3） |
| 工具参数错误重试轮数 | 未测 | 中位数 ≤1 |
| 运维排障 | doctor only | inspect 一次输出完整上下文 |

---

## 12. 参考

- [hermes.md](./hermes.md) — Hermes loop 差异与缺口  
- [grok-build.md](./grok-build.md) — Grok harness 可借鉴项  
- [../comparison.md](../comparison.md) — 三方速查  
- [../../architecture/layers/L4-runtime/agent-loop.md](../../architecture/layers/L4-runtime/agent-loop.md) — 实现说明  
- [../../../deploy/hermes-parity-roadmap.md](../../../deploy/hermes-parity-roadmap.md) — 历史 P1–P8 方法论（可复用 Phase 交付节奏）
