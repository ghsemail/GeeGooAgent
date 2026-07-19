# Agent Loop 功能验收指南

> 更新：2026-07-19。适用于本机或服务器（`~/.geegoo/bin/geegoo`）。  
> 离线卡片不依赖 LLM/MCP；端到端用例需要有效 `config.json` 与网络。

## 1. 一键离线验收（推荐先做）

```bash
export GEEGOO_CONFIG=~/.geegoo/config.json   # Windows: $env:GEEGOO_CONFIG=...

# 10 项 parity 卡片（clarify / delegate / schema / plan gate 等）
geegoo verify agent-loop

# 配置 + loop 参数 + 内嵌上述卡片
geegoo inspect --quick

# 连通性 + 抽样 tool 探针
geegoo doctor
```

**期望**：`verify agent-loop` 全部 `PASS`（退出码 0）；`doctor` 全绿或仅 `get_mcp_analysis` / `get_stock_daily_reports` 等非交易时段 WARN。

---

## 2. 功能对照表

| 功能 | 怎么验 | 期望 |
|------|--------|------|
| `geegoo inspect` | `geegoo inspect` | 打印 profile、loop 参数、toolsets、skills |
| NDJSON 事件流 | 见 §3 | `turn_start` + `turn_complete`，`schema_version: 1` |
| Plan 门控 | 见 §4 | 写操作先挂起，`y` 后执行 |
| HTTP clarify | 见 §5 | SSE 出现 `clarify`，POST 后续跑 |
| `delegate_tasks` | 见 §6 | 并行子任务，受 `delegate_max_parallel` 限制 |
| Schema 校验 | 见 §7 | 缺参返回 `参数校验失败` |
| Hooks | 见 §8 | 脚本收到 JSON stdin |
| 压缩血缘 | 见 §9 | `/session` 或 `inspect --session` 可见 chain |

---

## 3. NDJSON（Headless / CI）

```bash
geegoo chat --message "腾讯代码是多少" --output-format ndjson --cli 2>/dev/null | head
```

**期望**：每行合法 JSON，含 `"schema_version":1`；最后一轮有 `"event":"turn_complete"` 且 `data.assistant_text` 非空。

单元测试（开发机）：

```bash
go test ./internal/cli/chatrepl/... -run NDJSON -v
```

---

## 4. Plan 门控（写操作先确认）

**前提**：`config.json` 中 `plan_gate` 为 true（默认）。

### TUI

```bash
geegoo chat
```

输入会触发 mutating 工具的请求，例如：「帮我创建一个 DCA bot …」（需模型实际调用 `create_dca_bot`）。

**期望**：

1. 出现 `plan_proposed` / 计划说明，**不会**立刻创建 bot
2. 输入 `y` 或 `确认` → 执行写操作
3. 输入 `n` 或 `取消` → 提示已取消

### 单元测试

```bash
go test ./internal/agent/... -run PlanGate -v
go test ./internal/runtimeapi/... -run PlanHTTP -v
go test ./internal/chatsession/... -run PendingPlan -v
```

### HTTP plan 确认

1. `POST /v1/chat/stream` 触发 mutating 工具 → `turn_end` 含 `"plan_pending":true`
2. `POST /v1/chat/plan`：

```json
{ "session_id": "<id>", "approve": true }
```

3. 响应 `plan_pending` 应为 false，写操作已执行

协议详见 [runtime-clarify.md](./runtime-clarify.md) §Plan 门控。

---

## 5. HTTP clarify（agent-runtime）

协议说明：[runtime-clarify.md](./runtime-clarify.md)

### 自动化 E2E

```bash
go test ./internal/runtimeapi/... -run ClarifyHTTPE2E -v
```

### 手工（流式）

1. `POST /v1/chat/completions`，`stream: true`，消息里让模型需要澄清（或触发 `clarify` 工具）
2. 读 SSE，等待 `object: geegoo.agent_event` 且 `data.event == "clarify"`
3. `POST /v1/chat/clarify`：

```json
{ "session_id": "<SSE 里的 session_id>", "answer": "A股" }
```

4. 同一流应继续输出直至 `[DONE]`

---

## 6. 并行子 Agent

**配置**（可选）：

```json
{
  "delegate_max_parallel": 3
}
```

在 chat 中让模型调用 `delegate_tasks`，传入多个独立调研任务。

**期望**：`geegoo inspect` 显示 `delegate_max_parallel`；子任务并行完成（verbose 可见 `subagent_*` 事件）。

离线：`verify agent-loop` 含 `delegate_tasks registered`。

---

## 7. 工具 Schema 校验

模型传入缺参的 mutating 工具时，应在 MCP 调用前失败：

```bash
go test ./internal/tools/... -run ValidateArguments -v
```

`verify agent-loop` 卡片：`tool schema validation: required args enforced`。

---

## 8. Hooks（审计脚本）

1. 复制示例脚本并赋予执行权限：

```bash
chmod +x scripts/hooks/audit-tool.example.sh
```

2. 写入 `config.json`：

```json
{
  "hooks": {
    "tool_before": ["/absolute/path/to/audit-tool.example.sh"],
    "tool_after": ["/absolute/path/to/audit-tool.example.sh"],
    "fail_closed": false,
    "timeout_sec": 5
  }
}
```

3. 运行一次会调工具的 chat（如 `search_code`）

**期望**：`~/.geegoo/hooks/audit.log` 追加 JSON 行；`geegoo inspect` 显示 `hooks: configured=true`。

```bash
go test ./internal/tools/... -run HookRunner -v
```

---

## 9. 压缩血缘（B3）

长会话触发压缩后：

### Chat 内

```
/session
```

**期望**：`lineage_gen>=1`，`chain>=1`，每行 `gN compress|hygiene parent=... msgs X→Y tokens ...`

### 按 session id 查看

```bash
geegoo inspect --session <chat-session-id>
```

**期望**：`[compaction chain]` 列出每次压缩记录。

### 单元测试

```bash
go test ./internal/agent/... -run Compress -v
go test ./internal/chatsession/... -run LineageChain -v
```

---

## 10. 服务器验收清单（部署后）

在 **119.45.16.112** 或任意已安装节点：

```bash
cd ~/.geegoo/geegoo-agent && git log -1 --oneline
geegoo doctor
geegoo inspect --quick
```

当前基线（2026-07-19）：`2746993` 起，`inspect --quick` 应为 **10/10 PASS**。

---

## 11. 相关文档

- [agent-loop.md](./agent-loop.md) — 实现与配置项
- [runtime-clarify.md](./runtime-clarify.md) — HTTP clarify 协议
- [../../benchmark/agent-loop/optimization-roadmap.md](../../benchmark/agent-loop/optimization-roadmap.md) — 路线图与待办
