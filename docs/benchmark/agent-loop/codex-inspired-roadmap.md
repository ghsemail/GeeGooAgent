# Codex 借鉴优化方案（P1–P3）

> **日期:** 2026-08-21  
> **状态:** 规划定稿，待按 Phase 落地  
> **上游:** [OpenAI Codex](https://github.com/openai/codex)（Rust `codex-rs`）  
> **关联:** [optimization-roadmap.md](./optimization-roadmap.md)（Hermes/Grok）、[comparison.md](../comparison.md)、[backlog.md](../../architecture/backlog.md)

## 0. 摘要

GeeGooAgent 在 ReAct、Plan gate、子 Agent、Hooks、NDJSON headless、`inspect` 上已与 Codex 同类 harness **基本对齐**。本方案聚焦 Codex **仍值得抄的 harness 工程化能力**，按 P1 / P2 / P3 排期，**不引入 Shell / Git / 文件编辑**。

| 优先级 | 项数 | 主题 |
|--------|------|------|
| **P1** | 5 | Context、事件协议、Headless、Context Profile（AGENTS.md）、MCP Server |
| **P2** | 7 | 子 Agent 模型、会话 fork、Tool policy、Hook 回灌、Schema、Cost、测试 harness |
| **P3** | 6 | ACP、OTel、后台任务 UI、协作角色、Cloud 对齐、Plugin（按需）

**原则（与 optimization-roadmap §1.2 一致）：**

- 强化 **Workflow + Supervisor + Evidence + verify**，不削弱金融差异化。
- Kernel 仍在 Go；不迁 ReAct 到 Python。
- 出站模型仍经 **Policy → Gateway**。

---

## 1. 已对齐（本方案不重复建设）

| 能力 | GeeGooAgent | Codex 对应 |
|------|-------------|------------|
| ReAct 主循环 | `internal/agent/loop.go` | `codex-core` agent loop |
| Plan / 写操作门控 | `plan_gate` + `/v1/chat/plan` | Plan mode / approvals |
| 并行子 Agent | `delegate_tasks` | multi-agent / subagents |
| Hooks（审计） | `config.hooks` + `HookRunner` | `codex-rs/hooks` |
| Headless NDJSON | `--message --output-format ndjson` | `codex exec` streaming JSON |
| 配置发现 | `geegoo inspect` + `doctor` | `inspect` / doctor |
| 上下文压缩 + cache 断点 | `compressor.go` + `cache.go` | context + compaction |
| MCP 客户端 | GeeGooBot :3120 | `rmcp-client` |
| 会话 SQLite | `chatsession` + FTS5 | `thread-store` / `rollout` |
| 离线 loop 验收 | `verify agent-loop --offline` | integration suite |

---

## 2. 明确不做（Codex 有、GeeGoo 拒绝）

| Codex 能力 | 原因 |
|-----------|------|
| Shell / `exec-server` / bash | 非 coding agent；金融域无需求 |
| `apply-patch` / 多文件编辑 | 与 MCP 金融工具无关 |
| `git-utils` / PR review | 同上 |
| Git worktree 子 Agent | 编码专用 |
| 18+ Provider | 维持 DeepSeek / OpenAI / Minimax |
| Plugin Marketplace | YAGNI；Skill manifest 够用 |
| 向量 Semantic Memory | 刻意不做；Recall + FTS5 够用 |

---

## 3. P1 — 高价值

### P1-1 Context Fragments 框架

**Codex 参考:** `codex-rs/core/src/context/`、`ContextualUserFragment`、AGENTS.md 上下文硬边界（单条 ≤10K、总量有界、禁止 rewrite history）。

**问题:** tool 输出、recall、working state、hook 输出直接拼 `llm.Message`，难控 token、难审计注入来源、压缩策略分散。

**方案:**

1. 新增 `internal/context/`：
   - `Fragment` 接口：`Kind()`、`Render()`、`TokenEstimate()`、`Priority()`（压缩时丢弃顺序）
   - 实现体：`ToolResultFragment`、`RecallFragment`、`WorkingStateFragment`、`HookInjectFragment`、`SystemRulesFragment`
2. `internal/prompt/composer.go`：每轮从 fragments 组装 **user-side 动态上下文**；system prompt 保持字节稳定（现有 cache 策略不变）。
3. 压缩时：`Compressor` 先 drop 低优先级 fragment，再走现有 message 截断；对齐 `alignBoundaryBackward` + `sanitizeToolPairs`。
4. 接近上下文上限时注入 **预算提醒 fragment**（为 P2-7 Cost 预留接口）。

**涉及文件:**

| 操作 | 路径 |
|------|------|
| 新增 | `internal/context/fragment.go`、`composer.go`、`budget_reminder.go` |
| 改 | `internal/prompt/compressor.go`、`internal/agent/loop.go`、`internal/memport/` adapter |
| 测 | `internal/context/*_test.go`、`verify agent-loop` 新增卡片 |

**验收:**

- [ ] 单测：fragment 优先级丢弃顺序可预测
- [ ] `geegoo inspect` 展示当前 fragment 类型计数（debug 块）
- [ ] 长会话 turn 后 system message hash 不变（cache 友好）
- [ ] NDJSON 事件可选 `context_fragment_applied`（kind + token_est）

**依赖:** 无（P1 其余项可并行，但 P2-4 Hook 回灌依赖本项）


---

### P1-2 Thread / Turn / Item 事件协议

**Codex 参考:** `codex app-server` — Thread → Turn → Item；`item/started`、`item/completed`、`turn/completed`。

**问题:** TUI、HTTP SSE、`EmitProgress`、Flutter `agent_turn_telemetry` 字段不完全一致；缺少稳定 item 类型枚举。

**方案:**

1. 定义 `internal/runtime/events/schema.go`：
   ```text
   schema_version: 1
   turn_id, session_id, seq
   item_type: user_message | reasoning | assistant_message | tool_call | tool_result
              | plan_proposal | clarify_prompt | budget_warning | turn_complete
   ```
2. `ProgressEncoder`（text | ndjson | sse）统一出口；HTTP SSE 与 CLI NDJSON 共用 encoder（延续 optimization-roadmap A1）。
3. `runtimeapi`：`POST /v1/chat/stream` SSE  payload 对齐 item 模型；保留旧字段 1 个版本 **兼容期**（deprecated 标注）。
4. **Turn 生命周期:** `turn_start` → items… → `turn_complete` | `turn_failed` | `turn_aborted`；BFF 断连后 persist（GeeGooBot `upstreamRequestContext` 已做，补 `turn_aborted` 与 runtime 侧一致）。
5. Flutter：`trading_app` `agent_turn_telemetry.dart` 映射表与 schema 对齐（文档 + 可选跟进 PR）。

**涉及文件:**

| 操作 | 路径 |
|------|------|
| 新增/改 | `internal/runtime/events/`、`internal/runtime/progress.go` |
| 改 | `internal/runtimeapi/chat_stream.go`、`internal/cli/progress/ndjson.go` |
| 文档 | `docs/api/runtime-events.md`（新） |

**验收:**

- [ ] 同一 turn：CLI NDJSON 与 HTTP SSE 的 `item_type` 集合一致
- [ ] 集成测试：解析流得到 `turn_complete` + `finish_reason`
- [ ] `geegoo verify agent-loop` 新增「事件 schema」卡片
- [ ] App 旧客户端不 crash（未知 item 忽略）

**依赖:** 可与 P1-1 并行


---

### P1-3 Headless `exec` 子命令

**Codex 参考:** `codex exec`、`codex -p "..."`。

**问题:** CI/脚本需写长命令 `geegoo chat --message ... --output-format ndjson`。

**方案:**

1. 新增 `cmd/geegoo/exec.go`：
   ```bash
   geegoo exec -p "分析 00700.HK" [--session ID] [--output-format ndjson] [--config PATH]
   geegoo exec --message "..."   # 与 -p 等价
   ```
2. 内部调用现有 `chatrepl` 非 TTY 路径；默认 `output-format=ndjson`（exec 场景）。
3. 可选全局短选项：`geegoo -p "..."` 当 argv 解析到 `-p` 时转 exec（与 `main.go` 子命令不冲突）。

**涉及文件:** `cmd/geegoo/exec.go`、`cmd/geegoo/main.go`、`docs/README.md`

**验收:**

- [ ] `geegoo exec -p "ping"` 退出码 0 且 stdout 为合法 NDJSON
- [ ] `geegoo verify agent-loop` 文档引用 exec 示例
- [ ] 与 `scheduler` / systemd 脚本示例更新

**依赖:** P1-2 完成后 NDJSON 更稳定（可先薄包装）


---

### P1-4 Context Profile / AGENTS.md 规则加载

**Codex 参考:** 目录级 `AGENTS.md`、与 Cursor 生态互通。

**术语（避免与现有「workspace」混淆）:**

| 名称 | 含义 | 维度 |
|------|------|------|
| **WorkspaceRoot** | `output_dir`，报告/日志落盘 | 部署或用户数据目录（已有） |
| **StockWorkspace** | Workflow 运行态（code、bot_id…） | 单次 Skill 内按股+Bot（已有） |
| **Context Profile** | 可编辑 `AGENTS.md` 规则包 | **租户 × 上下文**（本项新增） |

Codex 的 workspace 是 **按项目目录**；GeeGoo 金融场景更自然的是 **按用户（Tenant）+ 可选按股票/ Bot/ 策略**。同一用户可有 **多个 Context Profile**，由 **Chat 会话** 选择激活哪些。

**问题:** Chat system 仅 Soul + 硬编码 ToolRouting；`rules/*.md` 是 Skill/verify 文档 SSOT，不进对话；用户无法按股/Bot 持久化分析偏好。

**概念模型:**

```text
Tenant（租户）     = user_id / mcp_token 隔离边界（与现有多租户一致）
Context Profile    = 一组可加载规则，type + key 唯一
Chat Session       = metadata 绑定 active_profiles[]（可多选，有上限）
```

**Profile 类型与路径（建议）:**

| type | key 示例 | AGENTS.md 路径 |
|------|----------|----------------|
| `user_default` | `{userId}` | `{GEEGOO_HOME}/tenants/{userId}/AGENTS.md` |
| `stock` | `00700.HK` | `{GEEGOO_HOME}/tenants/{userId}/stocks/{code}/AGENTS.md` |
| `bot` | `{botId}` | `{GEEGOO_HOME}/tenants/{userId}/bots/{botId}/AGENTS.md` |
| `strategy` | `dca-spcx` | `{GEEGOO_HOME}/tenants/{userId}/strategies/{slug}/AGENTS.md` |
| `global` | `default` | `{GEEGOO_HOME}/AGENTS.md`（可选，部署级） |

与 **SOUL.md** 分工：`SOUL.md` = 人设/沟通风格（同目录）；`AGENTS.md` = 业务偏好（默认市场、关注 Bot、回答长度等）。二者可并存。

**Prompt 加载顺序（Chat only）:**

```text
1. Soul（SOUL.md / defaultSoulText）
2. global AGENTS.md（若存在）
3. user_default AGENTS.md
4. 会话 active_profiles 中的 stock / bot / strategy（按固定优先级合并，总字节上限）
5. 硬规则（report-format、attitude-mapping 等 — 代码/embed，不可被 AGENTS 覆盖）
6. ToolRouting / MemoryRules / ServiceEndpoints
```

**硬规则不可覆盖:** AGENTS.md 不得削弱 Soul 中「禁止编造行情」等约束；盘前/盘中 Skill **不读** Context Profile（或仅从 working state 推导 stock/bot，见 P1-4+）；Supervisor + verify 仍为最后防线。

**分两档交付:**

| 档位 | 范围 | 多 Profile |
|------|------|------------|
| **P1-4 MVP** | `user_default` + 可选 `global`；`SystemForUser` 合并进 system | 每用户 1 个可编辑 AGENTS.md |
| **P1-4+** | 会话 `metadata.context_profiles`；API 创建/切换 profile；`stock`/`bot`/`strategy` 路径 | **每用户多个**，按会话切换 |

P1 实现须 **预留** 会话 metadata 字段与 `ResolveProfiles(session)` 接口，避免 P1-4+ 改表。

**配置（`config.json`）:**

```json
{
  "context_profiles": {
    "max_merged_bytes": 32768,
    "max_profiles_per_session": 4,
    "load_cursor_rules": false
  }
}
```

可选 `load_cursor_rules`：只读 `{profile_dir}/.cursor/rules/*.mdc` 文本，不执行 Cursor 逻辑。

**涉及文件:**

| 操作 | 路径 |
|------|------|
| 新增 | `internal/chatprompt/context_profile.go`、`profile_paths.go` |
| 改 | `internal/config/config.go`、`internal/chatprompt/prompt.go`（`SystemForUser`） |
| 改 | `internal/chatsession/store.go`（metadata 预留 `context_profiles`） |
| 改 | `internal/runtimeapi/chat_stream.go`（可选 body `context_profiles`） |
| 文档 | [rules-prompts.md](../../architecture/layers/L5-application/rules-prompts.md) |

**验收（MVP）:**

- [ ] 无 AGENTS.md 时行为与现网一致
- [ ] `tenants/{userId}/AGENTS.md` 存在时 chat system 增加对应段落
- [ ] `geegoo inspect` 展示已解析 profile 路径与合并字节数
- [ ] 恶意 AGENTS.md 不能跳过 report-format（Supervisor / verify 仍拦截）
- [ ] Skill workflow 行为不受 AGENTS.md 影响

**验收（P1-4+，可紧跟 MVP）:**

- [ ] 创建会话时可传 `context_profiles: ["stock:00700.HK"]`
- [ ] 从 App Bot 详情进 Chat 自动挂 `bot:{botId}`
- [ ] 多 profile 合并超 `max_merged_bytes` 时截断并 opslog 警告

**依赖:** P1-1 的 `SystemRulesFragment` 可简化 profile 合并；MVP 可先直接拼 `SystemBuilder`


---

### P1-5 MCP Server 模式（`geegoo mcp serve`）

**Codex 参考:** `codex-mcp`、`mcp-server` — 把 Agent 工具暴露给 Cursor/IDE。

**问题:** GeeGoo 仅 MCP 消费方；Cursor 无法直接调 `get_mcp_analysis` 等金融工具。

**方案:**

1. 新增 `cmd/geegoo/mcp.go` 子命令：`geegoo mcp serve [--config PATH] [--toolset chat|workflow]`
2. 新包 `internal/mcpserver/`：
   - stdio JSON-RPC MCP（`tools/list`、`tools/call`）
   - 从 `tools.Registry` 导出 schema（与 chat toolset 或子集一致）
   - 工具执行走现有 `runtime.Executor` + sandbox HTTP allowlist
   - 身份：`MCP_TOKEN` 环境变量或 config `mcp_token`（与 CLI 一致）
3. **不实现** MCP 资源/采样全量；仅 tools（MVP）。
4. 文档：Cursor `mcp.json` 配置示例。

**涉及文件:**

| 操作 | 路径 |
|------|------|
| 新增 | `internal/mcpserver/server.go`、`stdio.go`、`tools_adapter.go` |
| 新增 | `cmd/geegoo/mcp.go` |
| 文档 | `docs/engineering/mcp-server.md`（新） |

**验收:**

- [ ] Cursor MCP 列表可见 `search_code` 等工具
- [ ] 调用 `check_trading_day` 返回与直连 MCP 一致（dry-run 可测）
- [ ] 无 token 时 tools/call 明确 401 类错误
- [ ] `geegoo doctor` 可选 `--mcp-server-smoke`

**依赖:** 无


---

### P1 里程碑

| 检查项 | 命令 |
|--------|------|
| 单元 + 集成 | `go test ./...` |
| Loop 离线 | `geegoo verify agent-loop --offline`（扩展至 ≥14 项） |
| 生产冒烟 | `geegoo doctor`；`curl :3400/health` |
| 文档 | 本文件各 P1 验收勾选；更新 `implementation-status.md` |

---

## 4. P2 — 中等价值

### P2-1 子 Agent 指定模型

**Codex 参考:** `turn/start` model override；子 Agent 可选 model。

**方案:**

- `delegate_task` / `delegate_tasks` 参数增加可选 `model`、`provider`（或 `catalog_model_id`）。
- `SubAgent` 构造临时 `Gateway` 或 `Gateway.WithModelOverride`；受 `Policy` 白名单约束。
- synthesis / report 路径不受影响（独立 gateway）。

**落点:** `internal/agent/subagent.go`、`internal/llm/gateway.go`、`internal/tools/delegate.go`

**验收:**

- [ ] 主 Agent 用 fast 模型，delegate 用 strong 模型（配置 + 单测）
- [ ] 非法 model 回退主模型并 `tool_result` 带 warning

**依赖:** 无  

---

### P2-2 会话 Fork / Ephemeral Thread

**Codex 参考:** `thread/fork`、`ephemeral: true`。

**方案:**

- `POST /v1/sessions/{id}/fork` → 新 session_id，复制 messages 元数据（可选截断深度）。
- Chat 请求 body：`ephemeral: true` → 内存 session，不写 `chatsession` SQLite（或写 temp 表 TTL 清理）。
- App：「基于此分析新开对话」按钮对接 fork API。

**落点:** `internal/chatsession/`、`internal/runtimeapi/session_routes.go`

**验收:**

- [ ] fork 后两 session 独立；原 session 不变
- [ ] ephemeral 会话 `geegoo inspect --session` 标注 ephemeral
- [ ] 重启后 ephemeral 不存在

**依赖:** P1-2 事件协议（fork 产生新 turn_id）  

---

### P2-3 声明式 Tool Policy（金融版 execpolicy）

**Codex 参考:** `codex-rs/execpolicy` — prefix_rule、allow/prompt/forbidden。

**方案:**

- `config/tool_policy.yaml`（或 `~/.geegoo/tool_policy.yaml`）：
  ```yaml
  rules:
    - pattern: ["place_order"]
      decision: prompt
      justification: "实盘下单需确认"
    - pattern: ["delete_*"]
      decision: forbidden
  ```
- `internal/tools/policy/engine.go`：匹配 mutating 工具名；与 `approval.go` / `plan_gate` 合并决策（forbidden > prompt > allow）。
- 加载时校验 `match` / `not_match` 样例（Codex 风格单元测试）。

**落点:** `internal/tools/policy/`、`internal/config/config.go`

**验收:**

- [ ] forbidden 工具不进入 LLM 可见 schema（或调用即拒）
- [ ] `geegoo inspect` 展示 policy 规则数
- [ ] 与 `plan_gate=false` 组合行为文档化

**依赖:** 无  

---

### P2-4 Hook 输出回灌 Prompt

**Codex 参考:** `hook_additional_context` fragment。

**方案:**

- Hook 脚本 stdout 支持 JSON：`{"inject":"..."}` 或纯文本（≤2KB）。
- `HookRunner` 返回 `InjectContext []Fragment`；`tool_before` / `tool_after` 均可注入。
- 注入走 P1-1 fragment 管道，带 `source=hook` 标记。

**落点:** `internal/tools/hooks.go`、`internal/agent/tool_exec.go`

**验收:**

- [ ] 示例 hook 注入后下一轮 LLM 可见（dry-run 测）
- [ ] 超长 inject 截断 + opslog 警告

**依赖:** **P1-1**  

---

### P2-5 Runtime API OpenAPI 生成

**Codex 参考:** `generate-json-schema` / `generate-ts`。

**方案:**

- `geegoo runtimeapi generate-openapi --out docs/api/runtime-openapi.yaml`
- 从 `runtimeapi` 路由 + 请求/响应 struct 生成；CI `verify-openapi` 防 drift。
- GeeGooBot BFF 与 `trading_app` 可引用同一份 OpenAPI。

**落点:** `cmd/geegoo/runtimeapi.go`、`scripts/verify_openapi.sh`、`.github/workflows/ci.yml`

**验收:**

- [ ] 生成文件覆盖 `/v1/chat/stream`、`/v1/sessions/*`、fork（若 P2-2 已上线）
- [ ] CI 失败当 handler 无 schema 更新

**依赖:** P1-2、P2-2 路由稳定后生成更完整  

---

### P2-6 Token / Cost Manager

**Codex 参考:** `TokenBudgetContext`、`RolloutBudgetContext`。

**方案:**

- `internal/llm/cost.go`：每 session 累计 input/output tokens；可选 `config.budget.max_tokens_per_session`。
- 达 80% 注入 `BudgetWarningFragment`；达 100% 强制 `turn_complete`（summary 终局，与 loop 预算耗尽一致）。
- `GET /v1/metrics/overview` 与 Cockpit 展示 session 用量；backlog Cost Manager 落地。

**落点:** `internal/llm/cost.go`、`internal/runtimeapi/cockpit.go`、`internal/chatsession` metadata

**验收:**

- [ ] 超预算 turn 不继续调工具
- [ ] `geegoo inspect` 显示 budget 配置

**依赖:** P1-1 budget fragment  

---

### P2-7 Agent 集成测试 Harness

**Codex 参考:** `core/suite`、`test_codex`。

**方案:**

- `internal/agent/suite/harness.go`：`NewTestAgent(registry, gatewayFixture)` 
- 断言 NDJSON 事件序列：`turn_start` → `tool_*` → `turn_complete`
- 扩展 `geegoo verify agent-loop --offline` 至 20+ 卡片；关键路径进 CI。

**落点:** `internal/agent/suite/`、`cmd/geegoo/verify.go`

**验收:**

- [ ] CI 无网络跑 harness ≥5 场景
- [ ] 回归：plan_gate、delegate、clarify 各 1 用例

**依赖:** P1-2、P1-3  

---

### P2 里程碑

- OpenAPI 与 App/BFF 契约对齐
- Cost 指标可在 Cockpit 查看
- Tool policy + hook inject 生产可开关（config 默认保守）

---

## 5. P3 — 按需 / 低优先级

### P3-1 ACP / IDE 嵌入协议

**Codex 参考:** `codex app-server` JSON-RPC（Thread/Turn/Item）。

**方案:**

- 不重复造协议：P1-2 事件模型 **对齐 app-server 语义** 后，评估 [Agent Client Protocol](https://agentclientprotocol.com/) 或 Codex app-server 子集。
- 实现方式：stdio JSON-RPC 适配层，内部转 `runtimeapi` / `Agent.Run`。
- 与 backlog「IDE 扩展」合并；优先级低于 Flutter Dashboard。

**触发条件:** 需在 VS Code/Cursor 内嵌 GeeGoo 金融 Agent 且不愿走 MCP Server。


---

### P3-2 OTel 分布式追踪

**Codex 参考:** `codex-rs/otel`。

**方案:**

- 可选 `config.tracing.otel_endpoint`；span：turn、tool_call、gateway_chat、mcp_http。
- 默认关闭；与 backlog「分布式 Tracing」一致。

**触发条件:** 多租户 + 多机 runtime 排障频繁。


---

### P3-3 Scheduler 长任务 Watch UI

**Codex 参考:** Background tasks / Watchers。

**方案:**

- `geegoo scheduler watch` TUI：展示 cron 任务、最近 verdict、退避状态。
- 复用 `GET /v1/scheduler/status` + opslog。


---

### P3-4 协作角色模板（轻量 multi-agent）

**Codex 参考:** `multi_agent_role_instructions`。

**方案:**

- `config.chat.roles`：如 `analyst` / `risk_officer` 两套 system 追加段。
- Chat 可选 `/role risk`；**不**引入完整 multi-agent 进程。


---

### P3-5 Cloud Tasks 对齐（保持自托管）

**Codex 参考:** `cloud-tasks`。

**方案:**

- 维持 `geegoo scheduler` + systemd timer 为主。
- 可选：Webhook 触发 skill（backlog 已有）与 cloud 通知对接，不引入 Codex Cloud 依赖。


---

### P3-6 Plugin / Marketplace

**Codex 参考:** `core-plugins`、`connectors`。

**方案:**

- **暂不实现**；Skill manifest + `geegoo mcp serve` 覆盖扩展需求。
- 若未来需要：第三方 tool 注册走 MCP Server 而非内置 Plugin ABI。

**状态:** 明确 YAGNI，仅保留架构占位说明。

---

## 6. 依赖关系图

```text
P1-1 Context Fragments ──────► P2-4 Hook inject
         │                              │
         └──────────────────────► P2-6 Cost / Budget

P1-2 Event schema ─────────────► P2-2 Fork
         │                              │
         └──────────────────────► P2-5 OpenAPI
         └──────────────────────► P2-7 Test harness
         └──────────────────────► P3-1 ACP（可选）

P1-3 exec ─────────────────────► P2-7 CI 脚本

P1-4 Context Profile ──► P1-1 SystemRulesFragment（推荐顺序）
         │
         └── P1-4+ 会话 metadata / stock·bot profile（依赖 MVP 预留）

P1-5 MCP Server ── 独立，可与 P1 并行

P2-3 Tool policy ── 独立
P2-1 SubAgent model ── 独立
```

---

## 7. 推荐落地顺序

按依赖关系分批推进（无日历排期；可并行无依赖项）：

| 批次 | 交付 |
|------|------|
| **A** | P1-3 exec；P1-4 MVP（user AGENTS.md）；P1-2 schema 草案 + encoder |
| **B** | P1-1 Context Fragments MVP；P1-2 SSE/NDJSON 对齐；P1-4+ profile 切换（可选） |
| **C** | P1-5 MCP Server；P1 里程碑验收 |
| **D** | P2-1 子 Agent 模型；P2-3 Tool policy；P2-4 Hook inject |
| **E** | P2-2 Fork/Ephemeral；P2-5 OpenAPI |
| **F** | P2-6 Cost；P2-7 Harness + CI |
| **G** | P2 里程碑；文档与 `implementation-status` 同步 |
| **按需** | P3-1～P3-6 按触发条件启动 |


---

## 8. 验收总表

| ID | 验收命令 / 标准 | Phase |
|----|-----------------|-------|
| A1 | `go test ./internal/context/...` | P1-1 |
| A2 | CLI 与 HTTP 事件 `item_type` 一致 | P1-2 |
| A3 | `geegoo exec -p` NDJSON 合法 | P1-3 |
| A4 | Context Profile MVP：user AGENTS.md 加载且不削弱 report-format | P1-4 |
| A4+ | 会话 `context_profiles` 多 profile 合并 | P1-4+ |
| A5 | Cursor MCP 调 `search_code` 成功 | P1-5 |
| B1 | delegate 指定 model 单测通过 | P2-1 |
| B2 | `POST /v1/sessions/{id}/fork` | P2-2 |
| B3 | `tool_policy.yaml` forbidden 生效 | P2-3 |
| B4 | hook stdout inject 进入上下文 | P2-4 |
| B5 | `generate-openapi` CI 无 drift | P2-5 |
| B6 | 超 token 预算停止调工具 | P2-6 |
| B7 | `verify agent-loop` ≥20 卡片 | P2-7 |

---

## 9. 文档与 backlog 同步约定

1. 每项 **合并 main** 后：更新 [implementation-status.md](../../architecture/implementation-status.md)。
2. 从 [backlog.md](../../architecture/backlog.md) 勾选或迁移：Cost Manager、Webhook、IDE、OTel。
3. 更新 [comparison.md](../comparison.md) 增加 Codex 列（可选）。
4. Flutter 契约变更：在 `trading_app` 提 PR 并链接 `docs/api/runtime-events.md`。

---

## 10. 参考链接

| 资源 | URL |
|------|-----|
| OpenAI Codex | https://github.com/openai/codex |
| Codex app-server | `codex-rs/app-server/README.md` |
| Codex execpolicy | `codex-rs/execpolicy/README.md` |
| Codex context | `codex-rs/core/src/context/mod.rs` |
| GeeGoo optimization-roadmap | [optimization-roadmap.md](./optimization-roadmap.md) |
